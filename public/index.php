<?php
use think\App;
// +----------------------------------------------------------------------
// | ThinkPHP [ WE CAN DO IT JUST THINK ]
// +----------------------------------------------------------------------
// | Copyright (c) 2006~2018 http://thinkphp.cn All rights reserved.
// +----------------------------------------------------------------------
// | Licensed ( http://www.apache.org/licenses/LICENSE-2.0 )
// +----------------------------------------------------------------------
// | Author: liu21st <liu21st@gmail.com>
// +----------------------------------------------------------------------

// [ 应用入口文件 ]

require __DIR__ . '/../vendor/autoload.php';

// 处理URL大小写问题
if (isset($_SERVER['PATH_INFO']) && strpos($_SERVER['PATH_INFO'], '/Admin/') !== false) {
    $_SERVER['PATH_INFO'] = str_replace('/Admin/', '/admin/', $_SERVER['PATH_INFO']);
}
if (isset($_SERVER['REQUEST_URI']) && strpos($_SERVER['REQUEST_URI'], '/Admin/') !== false) {
    $_SERVER['REQUEST_URI'] = str_replace('/Admin/', '/admin/', $_SERVER['REQUEST_URI']);
}

// 执行HTTP应用并响应
$http = (new App())->http;

$response = $http->run();

$response->send();

$http->end($response);


