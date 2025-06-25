@echo off
chcp 65001 >nul

echo 启动简化版前端服务（仅支持评分功能）...

REM 设置默认值，避免程序崩溃
set PRODUCT_CATALOG_SERVICE_ADDR=localhost:3550
set CURRENCY_SERVICE_ADDR=localhost:7000
set CART_SERVICE_ADDR=localhost:7070
set RECOMMENDATION_SERVICE_ADDR=localhost:8080
set SHIPPING_SERVICE_ADDR=localhost:50051
set CHECKOUT_SERVICE_ADDR=localhost:5050
set AD_SERVICE_ADDR=localhost:9555
set SHOPPING_ASSISTANT_SERVICE_ADDR=localhost:8081

REM 设置前端服务端口为8090，避免与评分服务的8080冲突
set PORT=8090

echo 环境变量已设置
echo 启动前端服务在端口 8090...
frontend.exe --rating_service_addr=localhost:8080

pause