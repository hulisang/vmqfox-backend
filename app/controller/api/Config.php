<?php
namespace app\controller\api;

use think\facade\Db;
use think\facade\Request;
use think\facade\Config as ThinkConfig;

class Config extends BaseController
{
    /**
     * 获取系统设置
     * @return \think\Response
     */
    public function get()
    {
        $settings = Db::name("setting")->select()->toArray();
        
        // 将设置转换为键值对
        $config = [];
        foreach ($settings as $item) {
            $config[$item['vkey']] = $item['vvalue'];
        }
        
        // 获取系统信息
        $sysInfo = $this->getSystemInfo();
        
        // 合并配置和系统信息
        $result = array_merge($config, $sysInfo);
        
        return $this->success($result);
    }
    
    /**
     * 保存系统设置
     * @return \think\Response
     */
    public function save()
    {
        $settings = Request::param();
        
        // 允许修改的配置项
        $allowKeys = [
            'user', 'pass', 'notifyUrl', 'returnUrl', 'key',
            'close', 'payQf', 'wxpay', 'zfbpay'
        ];
        
        // 过滤不允许的键
        $filteredSettings = array_filter($settings, function($key) use ($allowKeys) {
            return in_array($key, $allowKeys);
        }, ARRAY_FILTER_USE_KEY);
        
        // 更新配置
        foreach ($filteredSettings as $key => $value) {
            Db::name("setting")->where("vkey", $key)->update([
                "vvalue" => $value
            ]);
        }
        
        return $this->success(null, '保存成功');
    }
    
    /**
     * 获取系统状态
     * @return \think\Response
     */
    public function status()
    {
        // 获取心跳和支付状态
        $lastheart = Db::name("setting")->where("vkey", "lastheart")->find();
        $lastpay = Db::name("setting")->where("vkey", "lastpay")->find();
        $jkstate = Db::name("setting")->where("vkey", "jkstate")->find();
        
        // 今日订单统计
        $today = strtotime(date("Y-m-d"), time());
        
        $todayOrder = Db::name("pay_order")
            ->where("create_date", ">=", $today)
            ->where("create_date", "<=", $today + 86400)
            ->count();
            
        $todaySuccessOrder = Db::name("pay_order")
            ->where("state", ">=", 1)
            ->where("create_date", ">=", $today)
            ->where("create_date", "<=", $today + 86400)
            ->count();
            
        $todayCloseOrder = Db::name("pay_order")
            ->where("state", -1)
            ->where("create_date", ">=", $today)
            ->where("create_date", "<=", $today + 86400)
            ->count();
            
        $todayMoney = Db::name("pay_order")
            ->where("state", ">=", 1)
            ->where("create_date", ">=", $today)
            ->where("create_date", "<=", $today + 86400)
            ->sum("price");
            
        // 总订单统计
        $countOrder = Db::name("pay_order")->count();
        $countMoney = Db::name("pay_order")
            ->where("state", ">=", 1)
            ->sum("price");
            
        // 监控状态 - 实时计算并更新数据库
        $monitorStatus = 0; // 0-未知 1-正常 2-异常
        $heartTime = intval($lastheart['vvalue']);

        if ($heartTime > 0) {
            if (time() - $heartTime < 180) {
                $monitorStatus = 1;
                // 如果监控端正常，确保数据库状态为1
                if ($jkstate['vvalue'] != '1') {
                    Db::name("setting")->where("vkey", "jkstate")->update(["vvalue" => "1"]);
                }
            } else {
                $monitorStatus = 2;
                // 如果监控端异常，更新数据库状态为0
                if ($jkstate['vvalue'] != '0') {
                    Db::name("setting")->where("vkey", "jkstate")->update(["vvalue" => "0"]);
                }
            }
        } else {
            // 如果没有心跳记录，设置为未绑定状态
            if ($jkstate['vvalue'] != '-1') {
                Db::name("setting")->where("vkey", "jkstate")->update(["vvalue" => "-1"]);
            }
        }
        
        $result = [
            'monitorStatus' => $monitorStatus,
            'lastHeartTime' => $heartTime > 0 ? date('Y-m-d H:i:s', $heartTime) : '',
            'lastPayTime' => intval($lastpay['vvalue']) > 0 ? date('Y-m-d H:i:s', intval($lastpay['vvalue'])) : '',
            'jkState' => intval($jkstate['vvalue']),
            'todayOrder' => $todayOrder,
            'todaySuccessOrder' => $todaySuccessOrder,
            'todayCloseOrder' => $todayCloseOrder,
            'todayMoney' => round($todayMoney, 2),
            'countOrder' => $countOrder,
            'countMoney' => round($countMoney, 2)
        ];
        
        return $this->success($result);
    }
    
    /**
     * 获取系统设置
     * @return \think\Response
     */
    public function settings()
    {
        $user = Db::name("setting")->where("vkey","user")->find();
        $pass = Db::name("setting")->where("vkey","pass")->find();
        $notifyUrl = Db::name("setting")->where("vkey","notifyUrl")->find();
        $returnUrl = Db::name("setting")->where("vkey","returnUrl")->find();
        $key = Db::name("setting")->where("vkey","key")->find();
        $lastheart = Db::name("setting")->where("vkey","lastheart")->find();
        $lastpay = Db::name("setting")->where("vkey","lastpay")->find();
        $jkstate = Db::name("setting")->where("vkey","jkstate")->find();
        $close = Db::name("setting")->where("vkey","close")->find();
        $payQf = Db::name("setting")->where("vkey","payQf")->find();
        $wxpay = Db::name("setting")->where("vkey","wxpay")->find();
        $zfbpay = Db::name("setting")->where("vkey","zfbpay")->find();
        
        if ($key['vvalue'] == "") {
            $key['vvalue'] = md5(time());
            Db::name("setting")->where("vkey","key")->update([
                "vvalue" => $key['vvalue']
            ]);
        }

        $result = [
            "user" => $user['vvalue'],
            "pass" => $pass['vvalue'],
            "notifyUrl" => $notifyUrl['vvalue'],
            "returnUrl" => $returnUrl['vvalue'],
            "key" => $key['vvalue'],
            "lastheart" => $lastheart['vvalue'],
            "lastpay" => $lastpay['vvalue'],
            "jkstate" => $jkstate['vvalue'],
            "close" => $close['vvalue'],
            "payQf" => $payQf['vvalue'],
            "wxpay" => $wxpay['vvalue'],
            "zfbpay" => $zfbpay['vvalue'],
        ];
        
        return $this->success($result);
    }
    
    /**
     * 保存系统设置
     * @return \think\Response
     */
    public function updateSettings()
    {
        $params = Request::param();
        
        // 更新设置
        if (isset($params['user'])) {
            Db::name("setting")->where("vkey", "user")->update(["vvalue" => $params['user']]);
        }
        if (isset($params['pass'])) {
            Db::name("setting")->where("vkey", "pass")->update(["vvalue" => $params['pass']]);
        }
        if (isset($params['notifyUrl'])) {
            Db::name("setting")->where("vkey", "notifyUrl")->update(["vvalue" => $params['notifyUrl']]);
        }
        if (isset($params['returnUrl'])) {
            Db::name("setting")->where("vkey", "returnUrl")->update(["vvalue" => $params['returnUrl']]);
        }
        if (isset($params['key'])) {
            Db::name("setting")->where("vkey", "key")->update(["vvalue" => $params['key']]);
        }
        if (isset($params['close'])) {
            Db::name("setting")->where("vkey", "close")->update(["vvalue" => $params['close']]);
        }
        if (isset($params['payQf'])) {
            Db::name("setting")->where("vkey", "payQf")->update(["vvalue" => $params['payQf']]);
        }
        if (isset($params['wxpay'])) {
            Db::name("setting")->where("vkey", "wxpay")->update(["vvalue" => $params['wxpay']]);
        }
        if (isset($params['zfbpay'])) {
            Db::name("setting")->where("vkey", "zfbpay")->update(["vvalue" => $params['zfbpay']]);
        }
        
        return $this->success(null, '保存成功');
    }
    
    /**
     * 获取监控状态
     * @return \think\Response
     */
    public function monitor()
    {
        $jkstate = Db::name("setting")->where("vkey", "jkstate")->find();
        $lastheart = Db::name("setting")->where("vkey", "lastheart")->find();
        $lastpay = Db::name("setting")->where("vkey", "lastpay")->find();
        
        $result = [
            'jkstate' => $jkstate['vvalue'],
            'lastheart' => $lastheart['vvalue'],
            'lastpay' => $lastpay['vvalue']
        ];
        
        return $this->success($result);
    }
    
    /**
     * 更新监控参数
     * @return \think\Response
     */
    public function updateMonitor()
    {
        $jk = Request::param('jk');
        
        if ($jk !== null) {
            Db::name("setting")->where("vkey", "jkstate")->update([
                "vvalue" => $jk
            ]);
        }
        
        return $this->success(null, '设置成功');
    }
    
    /**
     * 获取系统信息
     * @return array 系统信息
     */
    private function getSystemInfo()
    {
        // 获取MySQL版本
        $v = Db::query("SELECT VERSION();");
        $mysqlVersion = $v[0]['VERSION()'];
        
        // 检查GD库
        $gdInfo = '';
        if (function_exists("gd_info")) {
            $gd = gd_info();
            $gdInfo = $gd["GD Version"];
        } else {
            $gdInfo = 'GD库未开启';
        }
        
        return [
            'phpVersion' => PHP_VERSION,
            'phpOs' => PHP_OS,
            'server' => $_SERVER['SERVER_SOFTWARE'],
            'mysqlVersion' => $mysqlVersion,
            'thinkphpVersion' => ThinkConfig::get('app.version'),
            'runTime' => $this->getRunTime(),
            'appVersion' => ThinkConfig::get('app.ver'),
            'gdInfo' => $gdInfo
        ];
    }
    
    /**
     * 获取系统运行时间
     * @return string 运行时间
     */
    private function getRunTime()
    {
        // 在Windows环境下直接返回PHP版本信息
        if (strtoupper(substr(PHP_OS, 0, 3)) === 'WIN') {
            return 'Windows环境 - PHP ' . PHP_VERSION;
        }
        
        $output = '';
        if (false === ($str = @file("/proc/uptime"))) {
            return 'Unknown';
        }
        
        $str = explode(" ", implode("", $str));
        $str = trim($str[0]);
        $min = $str / 60;
        $hours = $min / 60;
        $days = floor($hours / 24);
        $hours = floor($hours - ($days * 24));
        $min = floor($min - ($days * 60 * 24) - ($hours * 60));
        
        if ($days !== 0) $output .= $days . "天";
        if ($hours !== 0) $output .= $hours . "小时";
        if ($min !== 0) $output .= $min . "分钟";
        
        return $output;
    }
} 