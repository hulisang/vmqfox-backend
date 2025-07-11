<?php
namespace app\controller\admin;

use think\facade\Db;
use think\facade\Session;
use think\facade\Request;
use think\facade\Config;
use app\service\QrcodeServer;
use Zxing\QrReader;

class Index
{
    public function index()
    {
        return 'by:vone';
    }

    public function getReturn($code = 1,$msg = "成功",$data = null){
        return array("code"=>$code,"msg"=>$msg,"data"=>$data);
    }

    public function getMain(){
        if (!$this->checkAdminSession()){
            return json($this->getReturn(-1,"没有登录"));
        }
        $today = strtotime(date("Y-m-d"),time());

        $todayOrder = Db::name("pay_order")
            ->where("create_date >=".$today)
            ->where("create_date <=".($today+86400))
            ->count();

        $todaySuccessOrder = Db::name("pay_order")
            ->where("state >=1")
            ->where("create_date >=".$today)
            ->where("create_date <=".($today+86400))
            ->count();

        $todayCloseOrder = Db::name("pay_order")
            ->where("state",-1)
            ->where("create_date >=".$today)
            ->where("create_date <=".($today+86400))
            ->count();

        $todayMoney = Db::name("pay_order")
            ->where("state >=1")
            ->where("create_date >=".$today)
            ->where("create_date <=".($today+86400))
            ->sum("price");

        $countOrder = Db::name("pay_order")
            ->count();
        $countMoney = Db::name("pay_order")
            ->where("state >=1")
            ->sum("price");

        $v = Db::query("SELECT VERSION();");
        $v=$v[0]['VERSION()'];

        if(function_exists("gd_info")) {
            $gd_info = @gd_info();
            $gd = $gd_info["GD Version"];
        }else{
            $gd = '<font color="red">GD库未开启！</font>';
        }

        return json($this->getReturn(1,"成功",array(
            "todayOrder"=>$todayOrder,
            "todaySuccessOrder"=>$todaySuccessOrder,
            "todayCloseOrder"=>$todayCloseOrder,
            "todayMoney"=>round($todayMoney,2),
            "countOrder"=>$countOrder,
            "countMoney"=>round($countMoney),
            "PHP_VERSION"=>PHP_VERSION,
            "PHP_OS"=>PHP_OS,
            "SERVER"=>$_SERVER ['SERVER_SOFTWARE'],
            "MySql"=>$v,
            "Thinkphp"=>"v".Config::get('app.version'),
            "RunTime"=>$this->sys_uptime(),
            "ver"=>"v".Config::get("app.ver"),
            "gd"=>$gd,
        )));
    }

    private function sys_uptime() {
        // 在Windows环境下直接返回PHP版本信息
        if (strtoupper(substr(PHP_OS, 0, 3)) === 'WIN') {
            return 'Windows环境 - PHP ' . PHP_VERSION;
        }
        
        $output='';
        if (false === ($str = @file("/proc/uptime"))) return 'Unknown';
        $str = explode(" ", implode("", $str));
        $str = trim($str[0]);
        $min = $str / 60;
        $hours = $min / 60;
        $days = floor($hours / 24);
        $hours = floor($hours - ($days * 24));
        $min = floor($min - ($days * 60 * 24) - ($hours * 60));
        if ($days !== 0) $output .= $days."天";
        if ($hours !== 0) $output .= $hours."小时";
        if ($min !== 0) $output .= $min."分钟";
        return $output;
    }

    // 通用的Session验证方法
    private function checkAdminSession()
    {
        $sessionId = Session::getId();
        $hasAdmin = Session::has("admin");
        $cookieSessionId = $_COOKIE['PHPSESSID'] ?? '';
        
        // 尝试手动使用Cookie中的PHPSESSID
        if ($cookieSessionId && $cookieSessionId != $sessionId) {
            // 记录原始sessionId
            $originalSessionId = $sessionId;
            
            // 尝试使用Cookie中的sessionId
            Session::setId($cookieSessionId);
            Session::init();
            
            // 重新检查
            $hasAdmin = Session::has("admin");
        }
        
        return $hasAdmin;
    }

    public function checkUpdate(){
        if (!$this->checkAdminSession()){
            return json($this->getReturn(-1,"没有登录"));
        }
        
        // 设置超时时间为1秒，避免长时间等待
        $ch = curl_init();
        curl_setopt($ch, CURLOPT_URL, "https://raw.githubusercontent.com/szvone/vmqphp/master/ver");
        curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
        curl_setopt($ch, CURLOPT_SSL_VERIFYHOST, false);
        curl_setopt($ch, CURLOPT_FOLLOWLOCATION, true);
        curl_setopt($ch, CURLOPT_AUTOREFERER, true);
        curl_setopt($ch, CURLOPT_TIMEOUT, 1); // 设置超时时间为1秒
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        $ver = curl_exec($ch);
        curl_close($ch);
        
        if (!$ver) {
            return json($this->getReturn(0,"检查更新失败，请稍后再试"));
        }
        
        $ver = explode("|",$ver);

        if (sizeof($ver)==2 && $ver[0]!=Config::get("app.ver")){
            return json($this->getReturn(1,"[v".$ver[0]."已于".$ver[1]."发布]","https://github.com/szvone/vmqphp"));
        }else{
            return json($this->getReturn(0,"程序是最新版"));
        }
    }

    public function getSettings(){
        if (!$this->checkAdminSession()){
            return json($this->getReturn(-1,"没有登录"));
        }
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
        if ($key['vvalue']==""){
            $key['vvalue'] = md5(time());
            Db::name("setting")->where("vkey","key")->update(array(
                "vvalue"=>$key['vvalue']
            ));
        }

        return json($this->getReturn(1,"成功",array(
            "user"=>$user['vvalue'],
            "pass"=>$pass['vvalue'],
            "notifyUrl"=>$notifyUrl['vvalue'],
            "returnUrl"=>$returnUrl['vvalue'],
            "key"=>$key['vvalue'],
            "lastheart"=>$lastheart['vvalue'],
            "lastpay"=>$lastpay['vvalue'],
            "jkstate"=>$jkstate['vvalue'],
            "close"=>$close['vvalue'],
            "payQf"=>$payQf['vvalue'],
            "wxpay"=>$wxpay['vvalue'],
            "zfbpay"=>$zfbpay['vvalue'],
        )));
    }

    public function saveSetting(){
        if (!$this->checkAdminSession()){
            return json($this->getReturn(-1,"没有登录"));
        }
        Db::name("setting")->where("vkey","user")->update(array("vvalue"=>Request::param("user")));
        Db::name("setting")->where("vkey","pass")->update(array("vvalue"=>Request::param("pass")));
        Db::name("setting")->where("vkey","notifyUrl")->update(array("vvalue"=>Request::param("notifyUrl")));
        Db::name("setting")->where("vkey","returnUrl")->update(array("vvalue"=>Request::param("returnUrl")));
        Db::name("setting")->where("vkey","key")->update(array("vvalue"=>Request::param("key")));
        Db::name("setting")->where("vkey","close")->update(array("vvalue"=>Request::param("close")));
        Db::name("setting")->where("vkey","payQf")->update(array("vvalue"=>Request::param("payQf")));
        Db::name("setting")->where("vkey","wxpay")->update(array("vvalue"=>Request::param("wxpay")));
        Db::name("setting")->where("vkey","zfbpay")->update(array("vvalue"=>Request::param("zfbpay")));

        return json($this->getReturn());
    }

    public function addPayQrcode(){
        if (!$this->checkAdminSession()){
            return json($this->getReturn(-1,"没有登录"));
        }
        $db = Db::name("pay_qrcode")->insert(array(
            "type"=>Request::param("type"),
            "pay_url"=>Request::param("pay_url"),
            "price"=>Request::param("price"),
        ));
        return json($this->getReturn());
    }

    public function getPayQrcodes(){
        if (!$this->checkAdminSession()){
            return json($this->getReturn(-1,"没有登录"));
        }
        $page = Request::param("page");
        $size = Request::param("limit");
        $type = Request::param("type");

        $where = array();
        if ($type) {
            $where[] = array("type", "=", $type);
        }

        $count = Db::name("pay_qrcode")->where($where)->count();
        $list = Db::name("pay_qrcode")
            ->where($where)
            ->page($page, $size)
            ->order("id desc")
            ->select();

        return json(array(
            "code" => 0,
            "msg" => "",
            "count" => $count,
            "data" => $list
        ));
    }

    public function delPayQrcode(){
        if (!$this->checkAdminSession()){
            return json($this->getReturn(-1,"没有登录"));
        }
        Db::name("pay_qrcode")->where("id", Request::param("id"))->delete();
        return json($this->getReturn());
    }

    public function getOrders(){
        if (!$this->checkAdminSession()){
            return json($this->getReturn(-1,"没有登录"));
        }
        $page = Request::param("page");
        $size = Request::param("limit");
        $state = Request::param("state");

        $where = array();
        if ($state !== null && $state !== "") {
            $where[] = array("state", "=", $state);
        }

        $count = Db::name("pay_order")->where($where)->count();
        $list = Db::name("pay_order")
            ->where($where)
            ->page($page, $size)
            ->order("id desc")
            ->select();

        return json(array(
            "code" => 0,
            "msg" => "",
            "count" => $count,
            "data" => $list
        ));
    }

    public function delOrder(){
        if (!$this->checkAdminSession()){
            return json($this->getReturn(-1,"没有登录"));
        }
        Db::name("pay_order")->where("id", Request::param("id"))->delete();
        return json($this->getReturn());
    }

    public function setBd(){
        if (!$this->checkAdminSession()){
            return json($this->getReturn(-1,"没有登录"));
        }
        $id = Request::param("id");
        $state = Request::param("state");

        Db::name("pay_qrcode")->where("id", $id)->update(array(
            "state" => $state
        ));

        return json($this->getReturn());
    }

    public function delGqOrder(){
        if (!$this->checkAdminSession()){
            return json($this->getReturn(-1,"没有登录"));
        }
        $time = time() - 300;
        Db::name("pay_order")->where("state", 0)->where("create_date < " . $time)->delete();
        return json($this->getReturn());
    }

    public function delLastOrder(){
        if (!$this->checkAdminSession()){
            return json($this->getReturn(-1,"没有登录"));
        }
        $time = time() - 86400;
        Db::name("pay_order")->where("create_date < " . $time)->delete();
        return json($this->getReturn());
    }

    public function enQrcode($url = null){
        // 使用checkAdminSession方法进行验证
        if (!$this->checkAdminSession()){
            // 如果是直接访问二维码图片，不要返回错误，继续处理
            // 只有在管理后台使用时才需要验证
            $referer = Request::header('referer');
            if ($referer && (strpos($referer, '/admin/') !== false)) {
                return json($this->getReturn(-1,"没有登录"));
            }
        }
        
        // 如果没有通过路由参数传递url，则尝试从查询参数获取
        if ($url === null) {
            $url = Request::param("url");
        }
        
        if (empty($url)) {
            return json($this->getReturn(-1, "缺少URL参数"));
        }
        
        $qrcodeServer = new QrcodeServer();
        $qrcode = $qrcodeServer->createQrcode($url);
        
        // 直接输出二维码图像数据
        $dataUri = $qrcode;
        
        // 从data URI中提取图像数据
        if (preg_match('/^data:image\/(\w+);base64,(.+)$/', $dataUri, $matches)) {
            $imageType = $matches[1];
            $imageData = base64_decode($matches[2]);
            
            // 设置正确的Content-Type
            header('Content-Type: image/' . $imageType);
            
            // 输出图像数据
            return $imageData;
        }
        
        // 如果无法解析data URI，则返回JSON
        return json($this->getReturn(1, "成功", array(
            "qrcode" => $qrcode
        )));
    }

    public function ip() {
        return json($this->getReturn(1, "成功", array(
            "ip" => Request::ip()
        )));
    }

    function getCurl($url, $post = 0, $cookie = 0, $header = 0, $nobaody = 0)
    {
        $ch = curl_init();
        curl_setopt($ch, CURLOPT_URL, $url);
        curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
        curl_setopt($ch, CURLOPT_SSL_VERIFYHOST, false);
        curl_setopt($ch, CURLOPT_FOLLOWLOCATION, true);
        curl_setopt($ch, CURLOPT_AUTOREFERER, true);
        curl_setopt($ch, CURLOPT_TIMEOUT, 30);
        curl_setopt($ch, CURLOPT_HEADER, $header);
        curl_setopt($ch, CURLOPT_NOBODY, $nobaody);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        if ($post) {
            curl_setopt($ch, CURLOPT_POST, true);
            curl_setopt($ch, CURLOPT_POSTFIELDS, $post);
        }
        if ($cookie) {
            curl_setopt($ch, CURLOPT_COOKIE, $cookie);
        }
        $output = curl_exec($ch);
        curl_close($ch);
        return $output;
    }
} 