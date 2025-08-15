<?php
namespace app\controller\api;

use think\facade\Db;
use think\facade\Request;

class Monitor extends BaseController
{
    /**
     * 监控端心跳
     * @return \think\Response
     */
    public function heart()
    {
        // 获取请求参数
        $t = Request::param('t');
        $sign = Request::param('sign');

        if (!$t || !$sign) {
            return $this->error('缺少必要参数');
        }

        // 获取数据库中的密钥
        $dbKey = Db::name("setting")->where("vkey", "key")->find();
        $key = $dbKey['vvalue'];

        // 验证签名
        $_sign = $t . $key;
        if ($sign != md5($_sign)) {
            return $this->error('密钥错误---请检查配置数据！');
        }

        // 更新最后心跳时间
        Db::name("setting")->where("vkey", "lastheart")->update(["vvalue" => time()]);

        // 更新监控端状态
        Db::name("setting")->where("vkey", "jkstate")->update(["vvalue" => "1"]);

        return $this->success(null, '心跳更新成功');
    }

    /**
     * 兼容旧版心跳接口 appHeart
     * @return \think\response\Json
     */
    public function appHeart()
    {
        // 获取请求参数
        $t = Request::param('t');
        $sign = Request::param('sign');
        
        if (!$t || !$sign) {
            return json(["code" => -1, "msg" => "缺少必要参数"]);
        }
        
        // 获取数据库中的密钥
        $dbKey = Db::name("setting")->where("vkey", "key")->find();
        $key = $dbKey['vvalue'];
        
        // 增强兼容性，尝试多种签名验证方式
        $_sign = $t.$key;
        $sign1 = md5($_sign);
        $sign2 = md5((string)$t.$key);
        $sign3 = md5(trim($t).$key);
        $sign4 = md5($t.trim($key));
        
        // 如果任一签名匹配则通过验证
        if ($sign != $sign1 && $sign != $sign2 && $sign != $sign3 && $sign != $sign4) {
            return json(["code" => -1, "msg" => "密钥错误---请检查配置数据！"]);
        }
        
        // 更新最后心跳时间
        Db::name("setting")->where("vkey", "lastheart")->update(["vvalue" => time()]);
        
        // 更新监控端状态
        Db::name("setting")->where("vkey", "jkstate")->update(["vvalue" => "1"]);
        
        // 使用框架的json助手函数返回
        return json(["code" => 1, "msg" => "成功"]);
    }

    /**
     * 监控端推送通知
     * @return \think\Response
     */
    public function push()
    {
        // 关闭超时订单
        $this->closeEndOrder();

        // 获取请求参数
        $t = Request::param('t');
        $type = Request::param('type');
        $price = Request::param('price');
        $sign = Request::param('sign');

        if (!$t || !$type || !$price || !$sign) {
            return $this->error('缺少必要参数');
        }

        // 获取系统密钥并验证签名
        $systemKey = Db::name("setting")->where("vkey", "key")->value('vvalue');
        if (empty($systemKey)) {
            return $this->error('系统密钥未设置');
        }
        
        $_sign = $type . $price . $t . $systemKey;
        if ($sign != md5($_sign)) {
            return $this->error('签名校验不通过');
        }

        // 精确化金额
        $price = sprintf("%.2f", $price);

        // 更新最后支付时间
        Db::name("setting")->where("vkey", "lastpay")->update(["vvalue" => time()]);

        // 查找订单
        $order = Db::name("pay_order")
            ->where("really_price", $price)
            ->where("state", 0)
            ->where("type", $type)
            ->find();

        // 如果未找到，则记录为无订单转账
        if (!$order) {
            $data = [
                "close_date" => 0, "create_date" => time(), "is_auto" => 0,
                "notify_url" => "", "order_id" => "无订单转账-" . time(), "param" => "无订单转账",
                "pay_date" => 0, "pay_id" => "无订单转账-" . time(), "pay_url" => "",
                "price" => $price, "really_price" => $price, "return_url" => "",
                "state" => 1, "type" => $type
            ];
            Db::name("pay_order")->insert($data);
            return $this->success(null, '成功'); // 按旧版逻辑，记录后即返回成功
        }
        
        // 找到订单，同步处理
        try {
            // 删除临时价格记录
            Db::name("tmp_price")->where("oid", $order['order_id'])->delete();

            // 先更新订单状态为1（已支付）
            Db::name("pay_order")->where("id", $order['id'])->update([
                "state" => 1, "pay_date" => time(), "close_date" => time()
            ]);

            // 准备并发送异步通知 (逻辑来自旧版 appPush)
            $notifyUrl = $order['notify_url'];
            if (!empty($notifyUrl)) {
                $p = "payId=".$order['pay_id']."&param=".$order['param']."&type=".$order['type']."&price=".$order['price']."&reallyPrice=".$order['really_price'];
                $signStr = $order['pay_id'].$order['param'].$order['type'].$order['price'].$order['really_price'].$systemKey;
                $p = $p . "&sign=".md5($signStr);

                if (strpos($notifyUrl, "?") === false) {
                    $notifyUrl = $notifyUrl."?".$p;
                } else {
                    $notifyUrl = $notifyUrl."&".$p;
                }
                
                $re = $this->getCurl($notifyUrl); // 发送GET请求
                
                // 如果通知失败，则更新订单状态为2
                if ($re != "success") {
                    Db::name("pay_order")->where("id", $order['id'])->update(["state" => 2]);
                }
            }
            return $this->success(null, '订单支付成功');
        } catch (\Exception $e) {
            return $this->error('订单处理失败: ' . $e->getMessage(), null, 500);
        }
    }
    
    /**
     * 兼容旧版推送接口 appPush
     * @return \think\response\Json
     */
    public function appPush()
    {
        // 关闭超时订单
        $this->closeEndOrder();

        // 获取请求参数
        $t = Request::param('t');
        $type = Request::param('type');
        $price = Request::param('price');
        $sign = Request::param('sign');

        if (!$t || !$type || !$price || !$sign) {
            return json(["code" => -1, "msg" => "缺少必要参数"]);
        }

        // 获取系统密钥并验证签名
        $systemKey = Db::name("setting")->where("vkey", "key")->value('vvalue');
        if (empty($systemKey)) {
             return json(["code" => -1, "msg" => "系统密钥未设置"]);
        }
        
        $_sign = $type.$price.$t.$systemKey;
        $sign1 = md5($_sign);

        if ($sign != $sign1) {
            // 为兼容某些客户端可能存在的编码或空格问题，尝试更多组合
            $sign2 = md5((string)$type.(string)$price.(string)$t.$systemKey);
            $sign3 = md5(trim($type).trim($price).trim($t).trim($systemKey));
            if ($sign != $sign2 && $sign != $sign3) {
                 return json(["code" => -1, "msg" => "签名校验不通过"]);
            }
        }

        // 精确化金额
        $price = sprintf("%.2f", $price);

        // 更新最后支付时间
        Db::name("setting")->where("vkey", "lastpay")->update(["vvalue" => time()]);

        // 查找订单
        $order = Db::name("pay_order")
            ->where("really_price", $price)
            ->where("state", 0)
            ->where("type", $type)
            ->find();

        // 如果未找到，则记录为无订单转账
        if (!$order) {
            $data = [
                "close_date" => 0, "create_date" => time(), "is_auto" => 0,
                "notify_url" => "", "order_id" => "无订单转账-" . time(), "param" => "无订单转账",
                "pay_date" => 0, "pay_id" => "无订单转账-" . time(), "pay_url" => "",
                "price" => $price, "really_price" => $price, "return_url" => "",
                "state" => 1, "type" => $type
            ];
            Db::name("pay_order")->insert($data);
            return json(["code" => 1, "msg" => "成功"]); // 按旧版逻辑，记录后即返回成功
        }
        
        // 找到订单后的处理: 实现"即发即弃"
        // 1. 立即响应成功给监控端
        echo json_encode(["code" => 1, "msg" => "成功"], JSON_UNESCAPED_UNICODE);
        if(ob_get_level() > 0) ob_flush();
        flush();
        if (function_exists('fastcgi_finish_request')) {
            fastcgi_finish_request();
        } else {
            if (session_id()) session_write_close();
            ignore_user_abort(true);
        }
        
        // 2. 在后台继续执行后续任务
        try {
            // 删除临时价格记录
            Db::name("tmp_price")->where("oid", $order['order_id'])->delete();

            // 先更新订单状态为1（已支付）
            Db::name("pay_order")->where("id", $order['id'])->update([
                "state" => 1, "pay_date" => time(), "close_date" => time()
            ]);

            // 准备并发送异步通知
            $notifyUrl = $order['notify_url'];
            if (!empty($notifyUrl)) {
                $p = "payId=".$order['pay_id']."&param=".$order['param']."&type=".$order['type']."&price=".$order['price']."&reallyPrice=".$order['really_price'];
                $signStr = $order['pay_id'].$order['param'].$order['type'].$order['price'].$order['really_price'].$systemKey;
                $p = $p . "&sign=".md5($signStr);

                if (strpos($notifyUrl, "?") === false) {
                    $notifyUrl = $notifyUrl."?".$p;
                } else {
                    $notifyUrl = $notifyUrl."&".$p;
                }
                
                $re = $this->postCurl($notifyUrl, []); // 发送GET请求
                
                // 如果通知失败，则更新订单状态为2
                if ($re != "success") {
                    Db::name("pay_order")->where("id", $order['id'])->update(["state" => 2]);
                }
            }
        } catch (\Exception $e) {
            // 记录异常，但不再响应给客户端
            // error_log('appPush Error: ' . $e->getMessage());
        }
        
        // 终止脚本，防止后续有任何输出
        exit;
    }
    
    /**
     * 完成订单
     * @param array $order 订单信息
     * @param string $payId 支付ID
     * @return bool 是否完成成功
     */
    private function completeOrder($order, $payId)
    {
        // 更新订单状态
        Db::name("pay_order")->where("id", $order['id'])->update([
            "state" => 1,
            "pay_date" => time(),
            "pay_id" => $payId
        ]);
        
        // 删除临时价格
        Db::name("tmp_price")->where("oid", $order['order_id'])->delete();
        
        // 异步通知
        $this->orderNotify($order);
        
        return true;
    }
    
    /**
     * 订单通知
     * @param array $order 订单信息
     * @return bool 是否通知成功
     */
    private function orderNotify($order)
    {
        if (empty($order['notify_url'])) {
            return false;
        }
        
        // 获取密钥
        $setting = Db::name("setting")->where("vkey", "key")->find();
        $key = $setting ? $setting['vvalue'] : '';
        
        // 构建通知参数
        $params = [
            'payId' => $order['order_id'],
            'param' => $order['param'],
            'type' => $order['type'],
            'price' => $order['price'],
            'reallyPrice' => $order['really_price']
        ];
        
        // 计算签名
        $sign = md5("payId=" . $params['payId'] . "&param=" . $params['param'] . "&type=" . $params['type'] . "&price=" . $params['price'] . "&reallyPrice=" . $params['reallyPrice'] . "&key=" . $key);
        $params['sign'] = $sign;
        
        // 发送异步通知
        $result = $this->postCurl($order['notify_url'], $params);
        
        // 记录通知结果
        $log = [
            'url' => $order['notify_url'],
            'params' => json_encode($params),
            'response' => $result,
            'time' => date('Y-m-d H:i:s')
        ];
        
        // 可以将通知日志记录到文件或数据库
        
        return true;
    }
    
    /**
     * 发送GET请求
     * @param string $url 请求URL
     * @return string 响应结果
     */
    private function getCurl($url)
    {
        $ch = curl_init();
        curl_setopt($ch, CURLOPT_URL, $url);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, 1);
        curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
        curl_setopt($ch, CURLOPT_SSL_VERIFYHOST, false);
        curl_setopt($ch, CURLOPT_TIMEOUT, 10);
        $result = curl_exec($ch);
        curl_close($ch);
        return $result;
    }

    /**
     * 发送POST请求
     * @param string $url 请求URL
     * @param array $params 请求参数
     * @return string 响应结果
     */
    private function postCurl($url, $params)
    {
        $ch = curl_init();
        curl_setopt($ch, CURLOPT_URL, $url);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, 1);
        curl_setopt($ch, CURLOPT_POST, 1);
        curl_setopt($ch, CURLOPT_POSTFIELDS, $params);
        curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
        curl_setopt($ch, CURLOPT_SSL_VERIFYHOST, false);
        curl_setopt($ch, CURLOPT_TIMEOUT, 10);
        $result = curl_exec($ch);
        curl_close($ch);
        return $result;
    }
    
    /**
     * 关闭超时订单
     * @return bool
     */
    private function closeEndOrder()
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
} 