<?php
namespace app\controller\api;

use think\facade\Db;
use think\facade\Request;

class Menu extends BaseController
{
    /**
     * 获取菜单数据
     * @return \think\Response
     */
    public function index()
    {
        // 获取管理菜单
        $menu = [
            [
                'id' => 1,
                'name' => '首页',
                'path' => '/dashboard',
                'icon' => 'dashboard',
                'component' => 'dashboard/index',
                'meta' => [
                    'title' => '首页',
                    'icon' => 'dashboard'
                ]
            ],
            [
                'id' => 2,
                'name' => '订单管理',
                'path' => '/order',
                'icon' => 'shopping',
                'component' => 'order/index',
                'meta' => [
                    'title' => '订单管理',
                    'icon' => 'shopping'
                ]
            ],
            [
                'id' => 3,
                'name' => '二维码管理',
                'path' => '/qrcode',
                'icon' => 'qrcode',
                'component' => 'Layout',
                'meta' => [
                    'title' => '二维码管理',
                    'icon' => 'qrcode'
                ],
                'children' => [
                    [
                        'id' => 31,
                        'name' => '微信二维码',
                        'path' => 'wechat',
                        'component' => 'qrcode/wechat',
                        'meta' => [
                            'title' => '微信二维码',
                            'icon' => 'wechat'
                        ]
                    ],
                    [
                        'id' => 32,
                        'name' => '支付宝二维码',
                        'path' => 'alipay',
                        'component' => 'qrcode/alipay',
                        'meta' => [
                            'title' => '支付宝二维码',
                            'icon' => 'alipay'
                        ]
                    ]
                ]
            ],
            [
                'id' => 4,
                'name' => '系统设置',
                'path' => '/settings',
                'icon' => 'setting',
                'component' => 'Layout',
                'meta' => [
                    'title' => '系统设置',
                    'icon' => 'setting'
                ],
                'children' => [
                    [
                        'id' => 41,
                        'name' => '基本设置',
                        'path' => 'system',
                        'component' => 'settings/system/index',
                        'meta' => [
                            'title' => '基本设置',
                            'icon' => 'system'
                        ]
                    ],
                    [
                        'id' => 42,
                        'name' => '监控设置',
                        'path' => 'monitor',
                        'component' => 'settings/monitor/index',
                        'meta' => [
                            'title' => '监控设置',
                            'icon' => 'monitor'
                        ]
                    ]
                ]
            ]
        ];
        
        return $this->success($menu);
    }
} 