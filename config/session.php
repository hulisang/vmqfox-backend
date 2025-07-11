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

// +----------------------------------------------------------------------
// | 会话设置
// +----------------------------------------------------------------------

return [
    // 默认会话驱动
    'default' => env('session.driver', 'file'),

    // 会话驱动配置
    'stores'  => [
        'file' => [
            // 驱动方式
            'type'          => 'File',
            'auto_start'    => true,
            // 存储路径
            'path'          => '',
            // 过期时间（秒）
            'expire'        => 86400,
            // 前缀
            'prefix'        => '',
            // 使用传入的session_id
            'use_trans_sid' => true,
            // 不使用cookie
            'use_cookies'   => true,
            // 启用安全会话
            'secure'        => false,
            // Cookie参数
            'cookie_domain' => '',
            'cookie_path'   => '/',
            'http_only'     => true,
        ],
        // 其它驱动配置
    ],
];
