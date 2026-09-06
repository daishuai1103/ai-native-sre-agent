#!/usr/bin/env python3
"""
Train a LightGBM classifier for SRE alert routing and export to ONNX.

Usage:
    python train_router.py [--output model.onnx] [--samples 500]

Categories: cache, container, db, network, web
"""

import argparse
import json
import os
import random
import sys

random.seed(42)

# --- Keyword vocabulary for feature extraction ---
CATEGORIES = ["cache", "container", "db", "network", "web"]

KEYWORDS = {
    "cache": [
        "redis", "cache", "memcached", "memory", "缓存", "内存",
        "key", "eviction", "expire", "ttl", "hit rate", "miss",
        "oom", "out of memory", "swap", "redis-cli", "flushall",
    ],
    "container": [
        "container", "docker", "pod", "cpu", "memory", "mem", "内存",
        "restart", "crash", "exit", "status", "运行中", "容器",
        "image", "volume", "disk", "space", "资源",
    ],
    "db": [
        "postgres", "mysql", "db", "database", "数据库", "sql",
        "query", "slow", "connection", "连接数", "deadlock",
        "replication", "备份", "backup", "wal", "buffer",
    ],
    "network": [
        "network", "nginx", "latency", "timeout", "网络", "延迟",
        "connection refused", "502", "504", "dns", "port",
        "tcp", "udp", "丢包", "bandwidth", "带宽", "499",
    ],
    "web": [
        "netbox", "web", "http", "500", "404", "403", "页面",
        "gunicorn", "django", "wsgi", "静态资源", "api",
        "request", "响应", "interface", "接口",
    ],
}

SAMPLE_TEMPLATES = {
    "cache": [
        "Redis 内存使用率达到 {pct}%，超过阈值",
        "缓存命中率下降到 {pct}%",
        "Redis 连接数异常，当前 {num} 个连接",
        "memcached 响应延迟升高到 {num}ms",
        "Redis 主从同步延迟超过 {num}s",
        "缓存失效风暴，大量 key 同时过期",
        "Redis OOM 被 kill，请检查内存配置",
        "Redis RDB 持久化失败",
        "缓存穿透严重，数据库压力上升",
        "Redis 集群某节点下线",
    ],
    "container": [
        "容器 {name} CPU 使用率 {pct}%",
        "容器 {name} 内存不足，OOM 风险",
        "Docker 容器频繁重启，已重启 {num} 次",
        "容器 {name} 磁盘空间不足",
        "Pod {name} 状态异常，CrashLoopBackOff",
        "容器 {name} 退出码 137，被 OOM kill",
        "容器网络不通，无法访问外部服务",
        "Docker daemon 响应缓慢",
        "镜像拉取失败，容器无法启动",
        "容器 {name} 文件描述符耗尽",
    ],
    "db": [
        "PostgreSQL 慢查询增多，超过 {num} 条",
        "数据库连接数达到上限 {num}",
        "MySQL 主从延迟超过 {num}s",
        "数据库死锁检测到 {num} 个",
        "PostgreSQL WAL 日志堆积",
        "数据库备份失败，已连续失败 {num} 次",
        "SQL 注入攻击检测到异常请求",
        "数据库 CPU 使用率 {pct}%，查询性能下降",
        "表空间不足，数据库写入失败",
        "索引失效，查询性能下降 {pct}%",
    ],
    "network": [
        "Nginx 502 错误率上升到 {pct}%",
        "网络延迟增加，平均 {num}ms",
        "DNS 解析失败率上升",
        "TCP 连接数异常，达到 {num}",
        "端口 {num} 无法访问，Connection Refused",
        "网络丢包率 {pct}%",
        "Nginx 499 状态码增多",
        "SSL 证书即将过期",
        "带宽使用率达到 {pct}%",
        "负载均衡节点 {name} 下线",
    ],
    "web": [
        "NetBox Web 页面加载缓慢",
        "API 接口 500 错误率上升",
        "Gunicorn worker 进程异常退出",
        "静态资源 404 错误增多",
        "Django ORM 查询超时",
        "Web 服务器内存泄漏，RSS 持续增长",
        "用户登录失败率上升",
        "CSRF 验证失败率异常",
        "Session 过期过快",
        "文件上传失败，超过大小限制",
    ],
}

NAMES = ["netbox", "redis", "postgres", "nginx", "worker-1", "worker-2", "scheduler"]


def generate_samples(n_per_class: int = 100):
    """Generate synthetic alert samples."""
    samples = []
    for cat in CATEGORIES:
        templates = SAMPLE_TEMPLATES[cat]
        for _ in range(n_per_class):
            tpl = random.choice(templates)
            text = tpl.format(
                pct=random.randint(60, 99),
                num=random.randint(5, 500),
                name=random.choice(NAMES),
            )
            samples.append({"text": text, "label": cat})
    random.shuffle(samples)
    return samples


def extract_features(text: str, vocab: list) -> list:
    """Simple keyword-based feature extraction (bag of keywords)."""
    text_lower = text.lower()
    return [1.0 if kw.lower() in text_lower else 0.0 for kw in vocab]


def build_vocab() -> list:
    """Build vocabulary from all keywords."""
    vocab = []
    seen = set()
    for cat_kws in KEYWORDS.values():
        for kw in cat_kws:
            if kw not in seen:
                seen.add(kw)
                vocab.append(kw)
    return vocab


def train_lightgbm(samples, vocab):
    """Train LightGBM and return the model."""
    try:
        import lightgbm as lgb
    except ImportError:
        print("LightGBM not installed, skipping model training.", file=sys.stderr)
        return None

    X = [extract_features(s["text"], vocab) for s in samples]
    y = [CATEGORIES.index(s["label"]) for s in samples]

    # Split
    split = int(len(X) * 0.8)
    X_train, X_val = X[:split], X[split:]
    y_train, y_val = y[:split], y[split:]

    train_data = lgb.Dataset(X_train, label=y_train)
    val_data = lgb.Dataset(X_val, label=y_val, reference=train_data)

    params = {
        "objective": "multiclass",
        "num_class": len(CATEGORIES),
        "metric": "multi_logloss",
        "num_leaves": 15,
        "learning_rate": 0.1,
        "feature_fraction": 0.8,
        "verbose": -1,
    }

    model = lgb.train(
        params,
        train_data,
        num_boost_round=50,
        valid_sets=[val_data],
        callbacks=[lgb.early_stopping(5)],
    )

    # Evaluate
    preds = model.predict(X_val)
    pred_labels = [p.argmax() for p in preds]
    accuracy = sum(1 for p, t in zip(pred_labels, y_val) if p == t) / len(y_val)
    print(f"Validation accuracy: {accuracy:.4f}")

    return model


def export_onnx(model, vocab, output_path: str):
    """Export LightGBM model to ONNX format."""
    try:
        from onnxmltools import convert_lightgbm
        from onnxmltools.convert.common.data_types import FloatTensorType
    except ImportError:
        print("onnxmltools not installed, skipping ONNX export.", file=sys.stderr)
        return False

    initial_types = [("float_input", FloatTensorType([None, len(vocab)]))]
    onnx_model = convert_lightgbm(model, initial_types=initial_types)

    with open(output_path, "wb") as f:
        f.write(onnx_model.SerializeToString())

    print(f"ONNX model saved to {output_path}")
    return True


def export_vocab(vocab, output_path: str):
    """Export vocabulary as JSON for Go to use."""
    with open(output_path, "w", encoding="utf-8") as f:
        json.dump({
            "vocab": vocab,
            "categories": CATEGORIES,
        }, f, ensure_ascii=False, indent=2)
    print(f"Vocabulary saved to {output_path}")


def main():
    parser = argparse.ArgumentParser(description="Train SRE alert router model")
    parser.add_argument("--output", default="model.onnx", help="ONNX output path")
    parser.add_argument("--vocab-output", default="vocab.json", help="Vocab JSON output path")
    parser.add_argument("--samples", type=int, default=100, help="Samples per class")
    args = parser.parse_args()

    # Generate samples
    samples = generate_samples(args.samples)
    print(f"Generated {len(samples)} samples ({len(CATEGORIES)} categories)")

    vocab = build_vocab()
    print(f"Vocabulary size: {len(vocab)}")

    # Train
    model = train_lightgbm(samples, vocab)

    # Export vocab (always)
    output_dir = os.path.dirname(args.output) or "."
    export_vocab(vocab, os.path.join(output_dir, args.vocab_output))

    # Export ONNX (if possible)
    if model is not None:
        export_onnx(model, vocab, args.output)

    print("Done.")


if __name__ == "__main__":
    main()
