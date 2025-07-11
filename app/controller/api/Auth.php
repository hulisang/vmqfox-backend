<?php
namespace app\controller\api;

use think\facade\Db;
use think\facade\Request;
use think\facade\Session;

class Auth extends BaseController
{
    /**
     * 登录
     * @return \think\Response
     */
    public function login()
    {
        $username = Request::param('username');
        $password = Request::param('password');
        
        if (empty($username) || empty($password)) {
            return $this->error('用户名或密码不能为空');
        }
        
        // 获取系统设置的用户名和密码
        $userSetting = Db::name('setting')->where('vkey', 'user')->find();
        $passSetting = Db::name('setting')->where('vkey', 'pass')->find();
        $keySetting = Db::name('setting')->where('vkey', 'key')->find();
        
        if (!$userSetting || !$passSetting) {
            return $this->error('系统未配置用户名或密码');
        }
        
        // 验证用户名和密码
        if ($username != $userSetting['vvalue'] || $password != $passSetting['vvalue']) {
            return $this->error('用户名或密码错误');
        }
        
        // 生成Token
        $token = md5('vmqphp_' . $keySetting['vvalue']);
        
        // 设置Session
        Session::set('admin', $username);
        
        return $this->success([
            'accessToken' => $token,
            'username' => $username
        ]);
    }
    
    /**
     * 退出登录
     * @return \think\Response
     */
    public function logout()
    {
        Session::delete('admin');
        return $this->success(null, '退出成功');
    }
} 