<?php
// +----------------------------------------------------------------------
// | ThinkPHP [ WE CAN DO IT JUST THINK ]
// +----------------------------------------------------------------------
// | Copyright (c) 2006~2018 http://thinkphp.cn All rights reserved.
// +----------------------------------------------------------------------
// | Licensed ( http://www.apache.org/licenses/LICENSE-2.0 )
// +----------------------------------------------------------------------
// | Author: liu21st <liu21st@gmail.com>
// +----------------------------------------------------------------------
use think\facade\Route;

// 兼容前端环境检测接口
Route::any('index/index/getReturn', 'index/Index/getReturnApi');

// 默认路由规则
Route::get('/', 'index/Index/index');

// 前后端分离API路由
Route::group('api', function () {
    // 认证相关
    Route::post('auth/login', 'api/Auth/login');
    Route::post('auth/logout', 'api/Auth/logout');
    
    // 用户相关
    Route::get('user/info', 'api/User/info');
    Route::get('user/list', 'api/User/list');
    
    // 菜单相关
    Route::get('menu', 'api/Menu/index');
    
    // 订单相关
    Route::any('order/create', 'api/Order/create');
    Route::get('order/list', 'api/Order/list');
    Route::get('order/detail/:id', 'api/Order/detail');
    Route::post('order/close/:id', 'api/Order/close');
    Route::get('order/check/:id', 'api/Order/check');
    Route::get('order/get/:id', 'api/Order/get'); // 新增：获取订单详情（支付页面专用）
    Route::delete('order/:id', 'api/Order/delete');
    Route::post('order/expired', 'api/Order/closeExpired');
    Route::delete('order/last', 'api/Order/deleteLast');
         Route::get('order/return-url/:id', 'api/Order/generateReturnUrl'); // 恢复：用于支付成功时生成带签名的返回URL
    
    // 二维码相关
    Route::get('qrcode/list', 'api/Qrcode/list');
    Route::post('qrcode/add', 'api/Qrcode/add');
    Route::post('qrcode/parse', 'api/Qrcode/parse');
    Route::get('qrcode/wechat', 'api/Qrcode/wechat');
    Route::get('qrcode/alipay', 'api/Qrcode/alipay');
    Route::post('qrcode/wechat', 'api/Qrcode/addWechat');
    Route::post('qrcode/alipay', 'api/Qrcode/addAlipay');
    Route::delete('qrcode/wechat/:id', 'api/Qrcode/deleteWechat');
    Route::delete('qrcode/alipay/:id', 'api/Qrcode/deleteAlipay');
    Route::post('qrcode/bind/:id', 'api/Qrcode/bind');
    Route::get('qrcode/generate', 'api/Qrcode/generate');
    Route::delete('qrcode/:id', 'api/Qrcode/delete');
    
    // 系统配置
    Route::get('config/get', 'api/Config/get');
    Route::post('config/save', 'api/Config/save');
    Route::get('config/status', 'api/Config/status');
    Route::get('config/settings', 'api/Config/settings');
    Route::post('config/settings', 'api/Config/updateSettings');
    Route::get('config/monitor', 'api/Config/monitor');
    Route::post('config/monitor', 'api/Config/updateMonitor');
    
    // 监控相关
    Route::any('monitor/heart', 'api/Monitor/heart');
    Route::any('monitor/push', 'api/Monitor/push');
    
    // 管理员设置相关 - 兼容旧版API
    Route::post('admin/index/getSettings', 'admin/Index/getSettings');
    Route::post('admin/index/saveSetting', 'admin/Index/saveSetting');
})->middleware(\app\middleware\CORS::class);

// 需要认证的API路由组
Route::group('api', function () {
    // 已经通过中间件认证的路由，这里不需要定义具体路由
})->middleware([\app\middleware\CORS::class, \app\middleware\Auth::class]);

// 兼容旧版API，保留原有路由
// 保留原有API路由组
Route::group('api', function () {
    // 登录相关
    Route::post('login', 'index/Index/login');
    Route::any('getMenu', 'index/Index/getMenu');
    
    // 订单相关
    Route::any('createOrder', 'index/Index/createOrder');
    Route::get('getOrder', 'index/Index/getOrder');
    Route::get('checkOrder', 'index/Index/checkOrder');
    Route::post('closeOrder', 'index/Index/closeOrder');
    Route::get('getState', 'index/Index/getState');
    
    // 应用相关
    //Route::post('appHeart', 'index/Index/appHeart');
    //Route::post('appPush', 'index/Index/appPush');
});

// 兼容旧版后台路由
Route::group('admin', function () {
    Route::get('/', 'admin/Index/index');
    Route::any('getMain', 'admin/Index/getMain');
    Route::get('checkUpdate', 'admin/Index/checkUpdate');
    Route::get('getSettings', 'admin/Index/getSettings');
    Route::post('saveSetting', 'admin/Index/saveSetting');
    
    // 二维码管理
    Route::post('addPayQrcode', 'admin/Index/addPayQrcode');
    Route::get('getPayQrcodes', 'admin/Index/getPayQrcodes');
    Route::post('delPayQrcode', 'admin/Index/delPayQrcode');
    Route::post('setBd', 'admin/Index/setBd');
    Route::get('enQrcode/:url', 'admin/Index/enQrcode');
    
    // 订单管理
    Route::get('getOrders', 'admin/Index/getOrders');
    Route::post('delOrder', 'admin/Index/delOrder');
    Route::post('delGqOrder', 'admin/Index/delGqOrder');
    Route::post('delLastOrder', 'admin/Index/delLastOrder');
    
    // 其他
    Route::get('ip', 'admin/Index/ip');
});

// 添加自定义路由
Route::get('think', function () {
    return 'hello,ThinkPHP8!';
});

Route::get('hello/:name', 'index/hello');

// 兼容旧版路由
Route::any('login', 'index/Index/login');
Route::any('getMenu', 'index/Index/getMenu');
Route::any('createOrder', 'index/Index/createOrder');
Route::any('closeOrder', 'index/Index/closeOrder');
Route::any('checkOrder', 'index/Index/checkOrder');
Route::any('getOrder', 'index/Index/getOrder');

// 监控端兼容路由已移至 route.php
// Route::any('appHeart', 'api/Monitor/heart');
// Route::any('appPush', 'api/Monitor/push');

// 添加明确的Admin路由映射
Route::any('admin/index/checkUpdate', 'admin/Index/checkUpdate');
Route::any('admin/index/:action', 'admin/Index/:action');

// 添加兼容的enQrcode路由，支持查询参数形式
Route::get('enQrcode', 'admin/Index/enQrcode');

// 添加兼容的admin/index/路由，支持所有前端请求
Route::get('admin/index/getOrders', 'admin/Index/getOrders');
Route::post('admin/index/setBd', 'admin/Index/setBd');
Route::post('admin/index/delOrder', 'admin/Index/delOrder');
Route::post('admin/index/delGqOrder', 'admin/Index/delGqOrder');
Route::post('admin/index/delLastOrder', 'admin/Index/delLastOrder');
Route::get('admin/index/getPayQrcodes', 'admin/Index/getPayQrcodes');
Route::post('admin/index/delPayQrcode', 'admin/Index/delPayQrcode');
Route::post('admin/index/addPayQrcode', 'admin/Index/addPayQrcode');
Route::post('admin/index/saveSetting', 'admin/Index/saveSetting');
Route::get('admin/index/getSettings', 'admin/Index/getSettings'); 