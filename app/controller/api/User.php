<?php
namespace app\controller\api;

use think\facade\Db;
use think\facade\Config;

class User extends BaseController
{
    /**
     * 获取用户信息
     * @return \think\Response
     */
    public function info()
    {
        // 从设置表中获取管理员用户名
        $user = Db::name("setting")->where("vkey", "user")->find();
        
        if (!$user) {
            return $this->error('未找到用户信息');
        }
        
        // 构建用户信息
        $userInfo = [
            'userId' => 1,
            'username' => $user['vvalue'],
            'realName' => 'V免签管理员',
            'avatar' => '',
            'roles' => ['admin'],
            'permissions' => [
                'dashboard', 'order', 'qrcode', 'setting', 'monitor'
            ]
        ];
        
        return $this->success($userInfo);
    }
    
    /**
     * 获取用户列表 (模拟数据，实际V免签只有一个管理员)
     * @return \think\Response
     */
    public function list()
    {
        // 在V免签中，实际上只有一个管理员用户，这里为了兼容前端API，返回模拟列表
        $user = Db::name("setting")->where("vkey", "user")->find();
        
        $list = [];
        if ($user) {
            $list[] = [
                'userId' => 1,
                'username' => $user['vvalue'],
                'realName' => 'V免签管理员',
                'status' => 1,
                'createTime' => date('Y-m-d H:i:s')
            ];
        }
        
        $result = [
            'list' => $list,
            'total' => count($list)
        ];
        
        return $this->success($result);
    }
} 