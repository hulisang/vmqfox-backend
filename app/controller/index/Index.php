<?php
namespace app\controller\index;

use think\facade\Db;
use think\facade\Session;
use think\facade\Request;
use think\facade\Config;

class Index
{
    public function __construct()
    {
        // 检查tmp_price表是否存在，如果不存在则创建
        try {
            $exists = Db::query("SHOW TABLES LIKE 'tmp_price'");
            if (empty($exists)) {
                Db::execute("CREATE TABLE `tmp_price` (
                    `id` int(11) NOT NULL AUTO_INCREMENT,
                    `price` varchar(255) NOT NULL,
                    `oid` varchar(255) NOT NULL,
                    PRIMARY KEY (`id`),
                    UNIQUE KEY `price` (`price`)
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8;");
            }
        } catch (\Exception $e) {
            // 处理创建表异常
        }
    }

    public function index()
    {
        return json($this->getReturn(1, "VMQ API 服务正常运行", [
            'version' => '2.0',
            'service' => 'vmqfox-backend',
            'status' => 'running'
        ]));
    }

    // 关闭超时订单
    public function closeEndOrder()
    {
        // 从设置中获取订单关闭时间（分钟）
        $closeTimeSetting = Db::name('setting')->where('vkey', 'close')->value('vvalue');
        $minutes = intval($closeTimeSetting) > 0 ? intval($closeTimeSetting) : 5; // 默认为5分钟
        
        $time = time() - ($minutes * 60);
        
        $orders = Db::name("pay_order")
            ->where("state", 0)
            ->where("create_date", "<", $time)
            ->select();
            
        foreach($orders as $order) {
            // 更新订单状态
            Db::name("pay_order")
                ->where("order_id", $order['order_id'])
                ->update(["state" => -1, "close_date" => time()]);
                
            // 删除对应的tmp_price记录
            Db::name("tmp_price")
                ->where("oid", $order['order_id'])
                ->delete();
        }
        
        // 清理孤立的tmp_price记录
        $tmpPrices = Db::name("tmp_price")->select();
        foreach($tmpPrices as $tmp) {
            $exists = Db::name("pay_order")->where("order_id", $tmp['oid'])->find();
            if (!$exists) {
                Db::name("tmp_price")->where("oid", $tmp['oid'])->delete();
            }
        }
        
        return true;
    }

    // 工具函数，供业务调用
    public function getReturn($code = 1, $msg = "成功", $data = null)
    {
        return ["code" => $code, "msg" => $msg, "data" => $data];
    }

    // 专门用于前端环境检测的接口
    public function getReturnApi()
    {
        return json($this->getReturn());
    }

    //后台用户登录
    public function login()
    {
        $user = Request::param('user');
        $pass = Request::param('pass');

        $_user = Db::name("setting")->where("vkey", "user")->find();
        $_pass = Db::name("setting")->where("vkey", "pass")->find();

        if ($user != $_user["vvalue"]) {
            return json($this->getReturn(-1, "账号或密码错误"));
        }
        if ($pass != $_pass["vvalue"]) {
            return json($this->getReturn(-1, "账号或密码错误"));
        }
        
        Session::set("admin", 1);
        Session::save();
        
        // 设置Cookie，确保路径正确，并设置较长的过期时间
        setcookie('PHPSESSID', Session::getId(), time() + 86400, '/', '', false, false);
        
        return json($this->getReturn(1, "成功"));
    }

    //后台菜单
    public function getMenu()
    {
        // 检查是否登录
        $hasAdmin = Session::has("admin");
        $cookieSessionId = $_COOKIE['PHPSESSID'] ?? '';
        
        // 尝试手动使用Cookie中的PHPSESSID
        if ($cookieSessionId && $cookieSessionId != Session::getId()) {
            // 尝试使用Cookie中的sessionId
            Session::setId($cookieSessionId);
            Session::init();
            
            // 重新检查
            $hasAdmin = Session::has("admin");
        }
        
        // 恢复Session检查
        if (!$hasAdmin) {
            return json($this->getReturn(-1, "没有登录"));
        }

        $menu = array(
            array(
                "name" => "系统设置",
                "type" => "url",
                "url" => "admin/setting.html?t=" . time(),
            ),
            array(
                "name" => "监控端设置",
                "type" => "url",
                "url" => "admin/jk.html?t=" . time(),
            ),
            array(
                "name" => "微信二维码",
                "type" => "menu",
                "node" => array(
                    array(
                        "name" => "添加",
                        "type" => "url",
                        "url" => "admin/addwxqrcode.html?t=" . time(),
                    ),
                    array(
                        "name" => "管理",
                        "type" => "url",
                        "url" => "admin/wxqrcodelist.html?t=" . time(),
                    )
                ),
            ), array(
                "name" => "支付宝二维码",
                "type" => "menu",
                "node" => array(
                    array(
                        "name" => "添加",
                        "type" => "url",
                        "url" => "admin/addzfbqrcode.html?t=" . time(),
                    ),
                    array(
                        "name" => "管理",
                        "type" => "url",
                        "url" => "admin/zfbqrcodelist.html?t=" . time(),
                    )
                ),
            ), array(
                "name" => "订单列表",
                "type" => "url",
                "url" => "admin/orderlist.html?t=" . time(),
            ), array(
                "name" => "Api说明",
                "type" => "url",
                "url" => "api.html?t=" . time(),
            )
        );

        // 确保返回正确的JSON格式
        return json($menu);
    }

    //创建订单
    public function createOrder()
    {
        $this->closeEndOrder();

        $payId = Request::param('payId');
        if (!$payId || $payId == "") {
            return json($this->getReturn(-1, "请传入商户订单号"));
        }
        $type = Request::param('type');
        if (!$type || $type == "") {
            return json($this->getReturn(-1, "请传入支付方式=>1|微信 2|支付宝"));
        }
        if ($type != 1 && $type != 2) {
            return json($this->getReturn(-1, "支付方式错误=>1|微信 2|支付宝"));
        }

        $price = Request::param('price');
        if (!$price || $price == "") {
            return json($this->getReturn(-1, "请传入订单金额"));
        }
        if ($price <= 0) {
            return json($this->getReturn(-1, "订单金额必须大于0"));
        }

        $sign = Request::param('sign');
        if (!$sign || $sign == "") {
            return json($this->getReturn(-1, "请传入签名"));
        }

        $isHtml = Request::param('isHtml');
        if (!$isHtml || $isHtml == "") {
            $isHtml = 0;
        }
        $param = Request::param('param');
        if (!$param) {
            $param = "";
        }

        $res = Db::name("setting")->where("vkey", "key")->find();
        $key = $res['vvalue'];

        if (Request::param('notifyUrl')) {
            $notify_url = Request::param('notifyUrl');
        } else {
            $res = Db::name("setting")->where("vkey", "notifyUrl")->find();
            $notify_url = $res['vvalue'];
        }

        if (Request::param('returnUrl')) {
            $return_url = Request::param('returnUrl');
        } else {
            $res = Db::name("setting")->where("vkey", "returnUrl")->find();
            $return_url = $res['vvalue'];
        }

        $_sign = md5($payId . $param . $type . $price . $key);
        if ($sign != $_sign) {
            return json($this->getReturn(-1, "签名错误"));
        }

        $jkstate = Db::name("setting")->where("vkey", "jkstate")->find();
        $jkstate = $jkstate['vvalue'];
        if ($jkstate!="1"){
            return json($this->getReturn(-1, "监控端状态异常，请检查"));
        }

        $reallyPrice = bcmul($price ,100);

        $payQf = Db::name("setting")->where("vkey", "payQf")->find();
        $payQf = $payQf['vvalue'];

        // 生成订单号
        $orderId = date("YmdHms") . rand(1, 9) . rand(1, 9) . rand(1, 9) . rand(1, 9);

        // 使用tmp_price表避免金额冲突
        $ok = false;
        for ($i = 0; $i < 10; $i++) {
            $tmpPrice = $reallyPrice . "-" . $type;

            try {
                $row = Db::execute("INSERT IGNORE INTO tmp_price (price,oid) VALUES ('" . $tmpPrice . "','".$orderId."')");
                if ($row) {
                    $ok = true;
                    break;
                }
            } catch (\Exception $e) {
                // 处理异常情况
            }
            
            // 根据配置微调价格
            if ($payQf == 1) {
                $reallyPrice++;
            } else if ($payQf == 2) {
                $reallyPrice--;
            }
        }

        if (!$ok) {
            return json($this->getReturn(-1, "订单超出负荷，请稍后重试"));
        }

        // 将分转换回元
        $reallyPrice = bcdiv($reallyPrice, 100, 2);

        // 获取支付URL
        if ($type == 1) {
            $payUrl = Db::name("setting")->where("vkey", "wxpay")->find();
            $payUrl = $payUrl['vvalue'];
        } else {
            $payUrl = Db::name("setting")->where("vkey", "zfbpay")->find();
            $payUrl = $payUrl['vvalue'];
        }

        // 检查URL是否为空，而不是检查是否等于"1"
        if ($payUrl == "") {
            if ($type == 1) {
                return json($this->getReturn(-1, "微信支付未配置"));
            } else {
                return json($this->getReturn(-1, "支付宝支付未配置"));
            }
        }

        $order = Db::name("pay_order")->where("pay_id", $payId)->find();
        if ($order) {
            return json($this->getReturn(-1, "订单号已存在"));
        }

        // 尝试查找匹配的二维码，如果找到则使用，否则使用默认URL
        $isAuto = 1; // 默认为自动模式，使用setting中配置的URL
        $_payUrl = Db::name("pay_qrcode")
            ->where("price", $reallyPrice)
            ->where("type", $type)
            ->find();
        if ($_payUrl) {
            $payUrl = $_payUrl['pay_url'];
            $isAuto = 0; // 使用二维码表中的URL
        }

        // 插入订单数据
        Db::name("pay_order")->insert([
            "order_id" => $orderId,
            "pay_id" => $payId,
            "type" => $type,
            "price" => $price,
            "really_price" => $reallyPrice,
            "param" => $param,
            "notify_url" => $notify_url,
            "return_url" => $return_url,
            "pay_url" => $payUrl, // 保存支付URL
            "is_auto" => $isAuto, // 保存是否自动模式
            "state" => 0,
            "create_date" => time(),
            "close_date" => 0, // 添加close_date默认值
            "pay_date" => 0 // 添加pay_date默认值
        ]);

        // 根据是否为HTML模式返回不同的响应
        if ($isHtml == 1) {
            echo "<script>window.location.href = 'payPage/pay.html?orderId=" . $orderId . "'</script>";
            exit;
        } else {
            $time = Db::name("setting")->where("vkey", "close")->find();
            $data = [
                "payId" => $payId,
                "orderId" => $orderId,
                "payType" => $type,
                "price" => $price,
                "reallyPrice" => $reallyPrice,
                "payUrl" => $payUrl,
                "isAuto" => $isAuto,
                "state" => 0,
                "timeOut" => $time['vvalue'],
                "date" => time()
            ];
            return json($this->getReturn(1, "成功", $data));
        }
    }

    /**
     * 监控端心跳接口
     */
    public function appHeart()
    {
        // 获取请求参数
        $t = Request::param('t');
        
        // 获取数据库中的密钥
        $dbKey = Db::name("setting")->where("vkey", "key")->find();
        $key = $dbKey['vvalue'];
        
        // 增强兼容性，尝试多种签名验证方式
        $_sign = $t.$key;
        $sign = Request::param('sign');
        $sign1 = md5($_sign);
        $sign2 = md5((string)$t.$key);
        $sign3 = md5(trim($t).$key);
        $sign4 = md5($t.trim($key));
        
        // 如果任一签名匹配则通过验证
        if ($sign != $sign1 && $sign != $sign2 && $sign != $sign3 && $sign != $sign4) {
            return json($this->getReturn(-1, "密钥错误---请检查配置数据！"));
        }
        
        // 更新最后心跳时间
        Db::name("setting")->where("vkey", "lastheart")->update(["vvalue" => time()]);
        
        // 更新监控端状态
        Db::name("setting")->where("vkey", "jkstate")->update(["vvalue" => "1"]);
        
        // 使用与原版ThinkPHP 5.1完全一致的响应格式，使用默认参数调用getReturn
        return json($this->getReturn());  // 将返回 {"code":1,"msg":"成功"}
    }
    
    /**
     * 监控端推送订单状态
     */
    public function appPush()
    {
        // 关闭超时订单
        $this->closeEndOrder();

        // 获取密钥
        $res2 = Db::name("setting")->where("vkey", "key")->find();
        $key = $res2['vvalue'];
        
        // 获取必要参数，优先从POST获取，其次GET，再从RAW请求体解析
        $t = $this->getParam('t');
        $type = $this->getParam('type');
        $price = $this->getParam('price');
        $sign = $this->getParam('sign');
        
        if (!$t || !$type || !$price || !$sign) {
            return json($this->getReturn(-1, "缺少必要参数"));
        }
        
        // 尝试多种签名组合方式
        $_sign = $type.$price.$t.$key;
        $sign1 = md5($_sign);
        $sign2 = md5((string)$type.(string)$price.(string)$t.$key);
        $sign3 = md5(trim($type).trim($price).trim($t).trim($key));
        
        // 签名验证，允许多种计算方式
        if ($sign != $sign1 && $sign != $sign2 && $sign != $sign3) {
            return json($this->getReturn(-1, "签名校验不通过"));
        }

        // 更新最后支付时间
        Db::name("setting")->where("vkey", "lastpay")->update(["vvalue" => time()]);

        // 查找匹配的订单，使用模糊匹配增加兼容性
        $res = Db::name("pay_order")
            ->where("really_price", $price)
            ->where("state", 0)
            ->where("type", $type)
            ->find();

        // 如果没有找到匹配订单
        if (!$res) {
            try {
                // 无匹配订单时记录为无订单转账
                $data = [
                    "close_date" => 0,
                    "create_date" => time(),
                    "is_auto" => 0,
                    "notify_url" => "",
                    "order_id" => "无订单转账-" . time(),  // 添加时间戳避免重复
                    "param" => "无订单转账",
                    "pay_date" => 0,
                    "pay_id" => "无订单转账-" . time(),  // 添加时间戳避免重复
                    "pay_url" => "",
                    "price" => $price,
                    "really_price" => $price,
                    "return_url" => "",
                    "state" => 1,
                    "type" => $type
                ];
    
                Db::name("pay_order")->insert($data);
                
                // 向客户端返回成功，但明确指出是无订单转账
                return json($this->getReturn(1, "无匹配订单，已记录为无订单转账"));
            } catch (\Exception $e) {
                // 返回错误信息给客户端
                return json($this->getReturn(-1, "记录无订单转账失败"));
            }
        }
        
        // 到这里说明已找到匹配订单，立即响应给客户端，避免连接超时
        // 这是关键：即发即弃（fire-and-forget）模式
        $this->asyncResponse($this->getReturn(1, "成功"));

        try {
            // 删除临时价格记录
            Db::name("tmp_price")
                ->where("oid", $res['order_id'])
                ->delete();

            // 更新订单状态
            Db::name("pay_order")->where("id", $res['id'])->update([
                "state" => 1,
                "pay_date" => time(),
                "close_date" => time()
            ]);

            // 准备通知参数
            $url = $res['notify_url'];
            $res2 = Db::name("setting")->where("vkey", "key")->find();
            $key = $res2['vvalue'];

            $p = "payId=".$res['pay_id']."&param=".$res['param']."&type=".$res['type']."&price=".$res['price']."&reallyPrice=".$res['really_price'];
            $sign = $res['pay_id'].$res['param'].$res['type'].$res['price'].$res['really_price'].$key;
            $p = $p . "&sign=".md5($sign);

            // 构建完整URL
            if (strpos($url, "?") === false) {
                $url = $url."?".$p;
            } else {
                $url = $url."&".$p;
            }
            
            // 发送通知
            $re = $this->getCurl($url);
            
            if ($re != "success") {
                // 通知失败时标记订单状态为2
                Db::name("pay_order")->where("id", $res['id'])->update(["state" => 2]);
            }
            
            // 异步模式下不返回任何内容
            return;
        } catch (\Exception $e) {
            // 异步模式下不返回任何错误信息，因为已经先返回成功了
            return;
        }
    }
    
    /**
     * 从请求中获取参数，优先POST，其次GET，再解析请求体
     */
    private function getParam($name)
    {
        // 优先从POST获取
        $value = Request::post($name);
        if ($value !== null) {
            return $value;
        }
        
        // 其次从GET获取
        $value = Request::get($name);
        if ($value !== null) {
            return $value;
        }
        
        // 最后尝试从原始请求体解析
        $raw = file_get_contents("php://input");
        $data = [];
        parse_str($raw, $data);
        return $data[$name] ?? null;
    }

    /**
     * 发送HTTP请求
     */
    private function getCurl($url, $post = 0, $cookie = 0, $header = 0, $nobaody = 0)
    {
        $ch = curl_init();
        curl_setopt($ch, CURLOPT_URL, $url);
        curl_setopt($ch, CURLOPT_TIMEOUT, 60);  // 与原版保持一致，最大超时时间为60秒
        curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
        curl_setopt($ch, CURLOPT_SSL_VERIFYHOST, false);
        
        if ($post) {
            curl_setopt($ch, CURLOPT_POST, 1);
            curl_setopt($ch, CURLOPT_POSTFIELDS, $post);
        }
        if ($header) {
            curl_setopt($ch, CURLOPT_HEADER, true);
        }
        if ($cookie) {
            curl_setopt($ch, CURLOPT_COOKIE, $cookie);
        }
        if ($nobaody) {
            curl_setopt($ch, CURLOPT_NOBODY, 1);
        }
        
        // 开启gzip压缩支持，与原版5.1代码保持一致
        curl_setopt($ch, CURLOPT_ENCODING, 'gzip');
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, 1);
        
        // 执行请求
        $ret = curl_exec($ch);
        
        curl_close($ch);
        return $ret;
    }
    
    /**
     * 异步响应，先返回结果给客户端，然后继续执行后续代码
     * 这可以避免"Software caused connection abort"错误
     * 
     * 特别为App端的请求优化，确保App能在短时间内收到响应，
     * 而服务端可以继续处理后续任务（如商户通知等）
     */
    private function asyncResponse($data)
    {
        try {
            // 发送数据
            if (is_array($data)) {
                echo json_encode($data, JSON_UNESCAPED_UNICODE);
            } else {
                echo $data;
            }
            
            // 确保所有输出缓冲区都被刷新
            if (ob_get_level() > 0) {
                ob_end_flush();
            }
            flush();
            
            // 在不同环境下选择合适的方法结束请求并继续处理
            if (function_exists('fastcgi_finish_request')) {
                // 最佳选择：PHP-FPM环境
                fastcgi_finish_request();
            } else {
                // 关闭会话写入，释放会话锁
                if (session_id()) {
                    session_write_close();
                }
                
                // 在某些服务器环境下，可能需要忽略客户端断开连接
                ignore_user_abort(true);
            }
        } catch (\Exception $e) {
            // 尝试确保响应已发送
            if (ob_get_level() > 0) {
                ob_end_flush();
            }
            flush();
        }
    }

    /**
     * 获取订单信息
     */
    public function getOrder()
    {
        $res = Db::name("pay_order")->where("order_id", Request::param("orderId"))->find();
        if ($res){
            $time = Db::name("setting")->where("vkey", "close")->find();

            $data = array(
                "payId" => $res['pay_id'],
                "orderId" => $res['order_id'],
                "payType" => $res['type'],
                "price" => $res['price'],
                "reallyPrice" => $res['really_price'],
                "payUrl" => $res['pay_url'],
                "isAuto" => $res['is_auto'],
                "state" => $res['state'],
                "timeOut" => $time['vvalue'],
                "date" => $res['create_date']
            );
            return json($this->getReturn(1, "成功", $data));
        }else{
            return json($this->getReturn(-1, "云端订单编号不存在"));
        }
    }
    
    /**
     * 查询订单状态
     */
    public function checkOrder()
    {
        $orderId = Request::param("orderId");
        if (!$orderId) {
            return json($this->getReturn(-1, "订单ID不能为空"));
        }
        
        // 查询订单
        $res = Db::name("pay_order")->where("order_id", $orderId)->find();
        
        if (!$res) {
            return json($this->getReturn(-1, "云端订单编号不存在"));
        }
        
        // 检查订单状态
        if ($res['state'] == 0) {
            return json($this->getReturn(-1, "订单未支付"));
        }
        
        if ($res['state'] == -1) {
            return json($this->getReturn(-1, "订单已过期"));
        }
        
        // 获取系统密钥
        $res2 = Db::name("setting")->where("vkey", "key")->find();
        $key = $res2['vvalue'];
        
        // 格式化价格，保证精度一致
        $res['price'] = number_format($res['price'], 2, ".", "");
        $res['really_price'] = number_format($res['really_price'], 2, ".", "");
        
        // 检查是否有回调URL
        if (empty($res['return_url'])) {
            return json($this->getReturn(1, "订单支付成功，但未设置回调URL"));
        }
        
        // 构建回调参数
        $p = "payId=".$res['pay_id']."&param=".$res['param']."&type=".$res['type']."&price=".$res['price']."&reallyPrice=".$res['really_price'];
        
        // 计算签名
        $sign = $res['pay_id'].$res['param'].$res['type'].$res['price'].$res['really_price'].$key;
        $p = $p . "&sign=".md5($sign);
        
        // 构建完整URL
        $url = $res['return_url'];
        if (strpos($url, "?") === false) {
            $url = $url."?".$p;
        } else {
            $url = $url."&".$p;
        }
        
        // 返回成功和跳转URL
        return json($this->getReturn(1, "成功", $url));
    }

    /**
     * 订单通知
     */
    private function orderNotify($order)
    {
        // 准备通知数据
        $data = [
            'payId' => $order['pay_id'],
            'param' => $order['param'],
            'type' => $order['type'],
            'price' => $order['price'],
            'reallyPrice' => $order['really_price']
        ];
        
        // 获取密钥
        $key = Db::name("setting")->where("vkey", "key")->find();
        
        // 计算签名
        $sign = md5($data['payId'] . $data['param'] . $data['type'] . $data['price'] . $data['reallyPrice'] . $key['vvalue']);
        $data['sign'] = $sign;
        
        // 发送通知
        $notifyResult = $this->getCurl($order['notify_url'], http_build_query($data));
        
        return $notifyResult;
    }
} 