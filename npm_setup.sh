#!/bin/bash
# setup.sh
rm -rf node_modules package-lock.json
npm install
chmod +x node_modules/.bin/vite
echo "✅ 依赖安装完成"