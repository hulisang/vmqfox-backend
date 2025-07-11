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

Route::get('think', function () {
    return 'hello,ThinkPHP5!';
});

Route::any('login','index/index/login');
Route::any('getMenu','index/index/getMenu');
Route::any('enQrcode','admin/index/enQrcode');
Route::any('createOrder','index/index/createOrder');


Route::any('getOrder','index/index/getOrder');
Route::any('checkOrder','index/index/checkOrder');
Route::any('getState','index/index/getState');

// 兼容旧版app的路由，指向新的专用方法
Route::any('appHeart','api/Monitor/appHeart');
Route::any('appPush','api/Monitor/appPush');

Route::any('closeEndOrder','index/index/closeEndOrder');

Route::get('getMain', 'admin/Index/getMain');

Route::any('index/index/getReturn', 'index/Index/getReturnApi');

return [

];
