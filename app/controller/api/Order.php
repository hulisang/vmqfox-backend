<?php
namespace app\controller\api;

use think\facade\Db;
use think\facade\Request;
use think\facade\Config;

class Order extends BaseController
{
    /**
     * 创建订单
     * @return \think\Response
     */
    public function create()
    {
        // 在创建新订单前，先关闭过期的订单
        $this->closeExpired();

        // 获取参数
        $payId = Request::param('payId');
        $param = Request::param('param');
        $type = Request::param('type');
        $price = Request::param('price');
        $sign = Request::param('sign');
        $notifyUrl = Request::param('notifyUrl');
        $returnUrl = Request::param('returnUrl');
        $isHtml = Request::param('isHtml', 0); // 添加isHtml参数支持，默认为0
        
        // 验证参数
        if (empty($payId) || empty($type) || empty($price) || empty($sign)) {
            return $this->error('参数不完整');
        }
        
        // 验证支付类型
        if ($type != 1 && $type != 2) {
            return $this->error('支付类型错误');
        }
        
        // 验证价格
        if (!is_numeric($price) || $price <= 0) {
            return $this->error('价格错误');
        }
        
        // 验证签名
        $key = Db::name('setting')->where('vkey', 'key')->value('vvalue');
        if (empty($key)) {
            return $this->error('系统未配置密钥');
        }
        
        // 兼容处理param为空的情况
        $paramValue = $param ?: '';
        $signStr = 'payId=' . $payId . '&param=' . $paramValue . '&type=' . $type . '&price=' . $price . '&key=' . $key;
        if (md5($signStr) != $sign) {
            return $this->error('签名错误');
        }

        // 检查监控端状态
        $jkstate = Db::name('setting')->where('vkey', 'jkstate')->value('vvalue');
        if ($jkstate != "1"){
            return $this->error('监控端状态异常，请检查');
        }
        
        // 检查商户订单号是否已存在
        $orderExists = Db::name('pay_order')->where('pay_id', $payId)->find();
        if ($orderExists) {
            return $this->error('商户订单号已存在，请勿重复提交');
        }
        
        // 生成订单号
        $orderId = date('YmdHis') . mt_rand(10000, 99999);
        
        // 使用分进行金额计算，避免精度问题
        $reallyPriceCent = bcmul($price, 100);
        
        // 获取payQf配置，用于金额递增/递减
        $payQf = Db::name('setting')->where('vkey', 'payQf')->value('vvalue');
        
        // 使用tmp_price表避免金额冲突
        $ok = false;
        for ($i = 0; $i < 10; $i++) {
            $tmpPrice = $reallyPriceCent . "-" . $type;
            try {
                $row = Db::execute("INSERT IGNORE INTO tmp_price (price, oid) VALUES (?, ?)", [$tmpPrice, $orderId]);
                if ($row) {
                    $ok = true;
                    break;
                }
            } catch (\Exception $e) {
                // 数据库异常
            }
            
            // 根据配置微调价格
            if ($payQf == '1') { // 金额递增模式
                $reallyPriceCent++;
            } else if ($payQf == '2') { // 金额递减模式
                $reallyPriceCent--;
            }
        }
        
        if (!$ok) {
            return $this->error('订单超出负荷，请稍后重试');
        }
        
        // 将分转换回元
        $reallyPrice = bcdiv($reallyPriceCent, 100, 2);
        
        $isAuto = 1; // 默认为自动模式 (is_auto=1)，需要用户手动输入金额
        $payUrl = '';
        
        // 1. 优先查找与浮动后金额完全匹配的二维码
        $qrcode = Db::name("pay_qrcode")
            ->where("price", $reallyPrice)
            ->where("type", $type)
            ->where("state", 0) // 确保二维码是启用的
            ->find();
            
        if ($qrcode) {
            // 找到匹配的二维码，使用固定的URL
            $payUrl = $qrcode['pay_url'];
            $isAuto = 0; // 标记为非自动模式 (is_auto=0)，扫码后金额固定
        } else {
            // 2. 如果没有找到，降级使用全局配置的通用收款码
            $settingKey = ($type == 1) ? 'wxpay' : 'zfbpay';
            $payUrl = Db::name('setting')->where('vkey', $settingKey)->value('vvalue');
        }
        
        // 检查最终是否有可用的支付URL
        if (empty($payUrl)) {
            $payMethodName = ($type == 1) ? '微信' : '支付宝';
            return $this->error("暂无可用支付二维码，请在后台【系统设置】或【{$payMethodName}二维码】中配置");
        }
        
        // 获取回调地址，如果请求中没有，则使用系统默认
        $finalNotifyUrl = $notifyUrl ?: Db::name('setting')->where('vkey', 'notifyUrl')->value('vvalue');
        $finalReturnUrl = $returnUrl ?: Db::name('setting')->where('vkey', 'returnUrl')->value('vvalue');
        
        // 保存订单信息
        $data = [
            'pay_id' => $payId,
            'order_id' => $orderId,
            'create_date' => time(),
            'type' => $type,
            'price' => $price,
            'really_price' => $reallyPrice,
            'state' => 0,
            'param' => $paramValue,
            'pay_url' => $payUrl,
            'is_auto' => $isAuto, // 增加 is_auto 字段
            'notify_url' => $finalNotifyUrl,
            'return_url' => $finalReturnUrl,
            'pay_date' => 0,
            'close_date' => 0 // 添加close_date字段，设为0
        ];
        
        $result = Db::name('pay_order')->insert($data);
        if (!$result) {
            return $this->error('创建订单失败');
        }
        
        // 如果isHtml=1，直接输出HTML跳转页面
        if ($isHtml == 1) {
            // 设置Content-Type为text/html
            header('Content-Type: text/html; charset=utf-8');
            
            // 返回HTML跳转脚本
            echo '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>正在跳转到支付页面...</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <style>
        body {
            background-color: #f5f5f5;
            font-family: Arial, sans-serif;
            text-align: center;
            padding-top: 100px;
        }
        .loading {
            display: inline-block;
            width: 50px;
            height: 50px;
            border: 3px solid rgba(0,0,0,.3);
            border-radius: 50%;
            border-top-color: #333;
            animation: spin 1s ease-in-out infinite;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        .text {
            margin-top: 20px;
            color: #333;
            font-size: 16px;
        }
    </style>
</head>
<body>
    <div class="loading"></div>
    <div class="text">正在跳转到支付页面，请稍候...</div>
    <script>
        window.location.href = "' . Config::get('app.frontend_url') . '/#/payment/' . $orderId . '";
    </script>
</body>
</html>';
            exit;
        }
        
        // 返回JSON格式的订单信息
        return $this->success([
            'payId' => $payId,
            'orderId' => $orderId,
            'payType' => $type,
            'price' => $price,
            'reallyPrice' => $reallyPrice,
            'payUrl' => $payUrl,
            'isAuto' => $isAuto,
            'redirectUrl' => Config::get('app.frontend_url') . '/#/payment/' . $orderId
        ]);
    }
    
    /**
     * 获取订单列表
     * @return \think\Response
     */
    public function list()
    {
        $page = Request::param('page', 1);
        $limit = Request::param('limit', 10);
        $state = Request::param('state');
        
        $where = [];
        if ($state !== null && $state !== '') {
            $where[] = ['state', '=', $state];
        }
        
        $count = Db::name('pay_order')->where($where)->count();
        $list = Db::name('pay_order')
            ->where($where)
            ->page($page, $limit)
            ->order('id desc')
            ->select()
            ->toArray();
            
        foreach ($list as &$item) {
            $item['create_time'] = date('Y-m-d H:i:s', $item['create_date']);
            $item['state_text'] = $this->getOrderStateText($item['state']);
            $item['type_text'] = $item['type'] == 1 ? '微信' : '支付宝';
        }
        
        return $this->success([
            'total' => $count,
            'items' => $list
        ]);
    }
    
    /**
     * 获取订单详情
     * @param int $id 订单ID
     * @return \think\Response
     */
    public function detail($id)
    {
        $order = Db::name('pay_order')->where('id', $id)->find();
        if (!$order) {
            return $this->error('订单不存在');
        }
        
        $order['create_time'] = date('Y-m-d H:i:s', $order['create_date']);
        $order['state_text'] = $this->getOrderStateText($order['state']);
        $order['type_text'] = $order['type'] == 1 ? '微信' : '支付宝';
        
        return $this->success($order);
    }
    
    /**
     * 获取订单详情，供支付页面使用
     * @param string $id 订单ID
     * @return \think\Response
     */
    public function get($id)
    {
        // 获取参数
        $orderId = $id;
        
        if (empty($orderId)) {
            return $this->error('订单号不能为空');
        }
        
        // 查询订单
        $order = Db::name('pay_order')->where('order_id', $orderId)->find();
        if (!$order) {
            return $this->error('订单不存在');
        }
        
        // 获取关闭时间
        $closeTimeSetting = Db::name('setting')->where('vkey', 'close')->value('vvalue');
        
        // 由于close配置项是订单超时时间，直接使用其值
        $closeTime = intval($closeTimeSetting) > 0 ? intval($closeTimeSetting) : 5; // 默认5分钟
        
        // 计算剩余时间
        $timeoutSeconds = $closeTime * 60;
        $elapsedSeconds = time() - $order['create_date'];
        $remainingSeconds = max(0, $timeoutSeconds - $elapsedSeconds);
        
        // 调试日志
        error_log("订单ID: {$orderId}, 超时时间配置原始值: {$closeTimeSetting}, 处理后的超时时间: {$closeTime}分钟, 剩余: {$remainingSeconds}秒");
        
        // 返回订单详情，适配前端支付页面所需的字段格式
        return $this->success([
            'payId' => $order['pay_id'],
            'orderId' => $order['order_id'],
            'payType' => $order['type'],
            'price' => floatval($order['price']),
            'reallyPrice' => floatval($order['really_price']),
            'payUrl' => $order['pay_url'],
            'isAuto' => $order['is_auto'],
            'state' => $order['state'],
            'stateText' => $this->getOrderStateText($order['state']),
            'timeOut' => $closeTime, // 这里传递的是分钟数，前端会乘以60转为秒
            'date' => $order['create_date'],
            'remainingSeconds' => $remainingSeconds, // 添加剩余秒数
            'return_url' => $order['return_url'], // 添加商户返回URL
            'param' => $order['param'] // 添加订单自定义参数
        ]);
    }
    
    /**
     * 关闭订单
     * @param int $id 订单ID
     * @return \think\Response
     */
    public function close($id)
    {
        $order = Db::name('pay_order')->where('id', $id)->find();
        if (!$order) {
            return $this->error('订单不存在');
        }
        
        if ($order['state'] != 0) {
            return $this->error('只能关闭未支付的订单');
        }
        
        $result = Db::name('pay_order')->where('id', $id)->update([
            'state' => -1,
            'close_date' => time()
        ]);
        
        if (!$result) {
            return $this->error('关闭订单失败');
        }
        
        // 成功关闭订单后，释放tmp_price中锁定的金额
        Db::name('tmp_price')->where('oid', $order['order_id'])->delete();

        return $this->success(null, '关闭订单成功');
    }
    
    /**
     * 检查订单状态
     * @param string $id 订单ID
     * @return \think\Response
     */
    public function check($id)
    {
        try {
            // 获取参数
            $orderId = $id;
            
            if (empty($orderId)) {
                error_log("订单ID为空");
                return $this->error('订单号不能为空');
            }
            
            // 查询订单
            $order = Db::name('pay_order')->where('order_id', $orderId)->find();
            if (!$order) {
                error_log("订单不存在: {$orderId}");
                return $this->error('订单不存在');
            }
    
            // 调试日志
            error_log("检查订单状态 - 订单ID: {$orderId}, 状态: {$order['state']}, 创建时间: " . date('Y-m-d H:i:s', $order['create_date']));
    
            // 获取订单超时时间设置
            $closeTimeSetting = Db::name('setting')->where('vkey', 'close')->value('vvalue');
            $closeTime = intval($closeTimeSetting) > 0 ? intval($closeTimeSetting) : 5; // 默认5分钟
            $timeoutSeconds = $closeTime * 60;
            $elapsedSeconds = time() - $order['create_date'];
            $remainingSeconds = max(0, $timeoutSeconds - $elapsedSeconds);
            
            error_log("订单时间详情 - 配置的超时时间: {$closeTime}分钟({$timeoutSeconds}秒), 已经过去: {$elapsedSeconds}秒, 剩余: {$remainingSeconds}秒");
            error_log("订单创建时间戳: {$order['create_date']}, 当前时间戳: " . time());
    
            // 根据订单状态返回不同指令
            switch ($order['state']) {
                case 1: // 已支付
                    // 返回跳转地址，前端收到后应立即跳转
                    error_log("订单已支付，返回跳转地址: {$order['return_url']}");
                    return $this->success([
                        'redirectUrl' => $order['return_url'], 
                        'remainingSeconds' => 0,
                        'return_url' => $order['return_url'],
                        'param' => $order['param']
                    ], '支付成功');
                    
                case -1: // 已关闭/过期
                    // 返回过期状态，前端收到后应显示过期信息
                    error_log("订单已过期");
                    return $this->success([
                        'state' => -1, 
                        'remainingSeconds' => 0,
                        'return_url' => $order['return_url'],
                        'param' => $order['param']
                    ], '订单已过期');
                    
                default: // 未支付
                    // 检查是否过期但状态未更新
                    if ($elapsedSeconds > $timeoutSeconds) {
                        // 订单已过期但状态未更新，立即更新状态
                        error_log("订单已过期但状态未更新，更新状态为已关闭 (已过去: {$elapsedSeconds}秒, 超时时间: {$timeoutSeconds}秒)");
                        
                        try {
                            Db::name('pay_order')->where('order_id', $orderId)->update([
                                'state' => -1,
                                'close_date' => time()
                            ]);
                            
                            // 同时释放tmp_price中锁定的金额
                            Db::name('tmp_price')->where('oid', $orderId)->delete();
                        } catch (\Exception $e) {
                            error_log("更新订单状态异常: " . $e->getMessage());
                            // 即使更新失败，仍然返回过期状态给前端
                        }
                        
                        return $this->success([
                            'state' => -1, 
                            'remainingSeconds' => 0,
                            'return_url' => $order['return_url'],
                            'param' => $order['param']
                        ], '订单已过期');
                    }
                    
                    // 返回未支付状态以及剩余时间，前端收到后应继续轮询
                    error_log("订单未支付，还有 {$remainingSeconds} 秒过期");
                    return $this->success([
                        'state' => 0, 
                        'remainingSeconds' => $remainingSeconds,
                        'return_url' => $order['return_url'],
                        'param' => $order['param']
                    ], '订单未支付');
            }
        } catch (\Exception $e) {
            // 记录错误日志
            error_log("检查订单状态异常: " . $e->getMessage() . "\n" . $e->getTraceAsString());
            
            // 返回友好的错误信息
            return $this->error('服务器处理请求时发生错误，请稍后重试');
        }
    }
    
    /**
     * 删除订单
     * @param int $id 订单ID
     * @return \think\Response
     */
    public function delete($id)
    {
        // 先查找订单以获取order_id，以便后续删除关联的tmp_price
        $order = Db::name('pay_order')->where('id', $id)->find();
        if (!$order) {
            // 订单本就不存在，可认为删除操作已达成
            return $this->success(null, '删除订单成功');
        }

        $result = Db::name('pay_order')->where('id', $id)->delete();
        if (!$result) {
            return $this->error('删除订单失败');
        }

        // 成功删除订单后，释放tmp_price中锁定的金额
        Db::name('tmp_price')->where('oid', $order['order_id'])->delete();

        return $this->success(null, '删除订单成功');
    }
    
    /**
     * 关闭过期订单
     * @return \think\Response
     */
    public function closeExpired()
    {
        // 从设置中获取订单关闭时间（分钟）
        $closeTimeSetting = Db::name('setting')->where('vkey', 'close')->value('vvalue');
        $minutes = intval($closeTimeSetting) > 0 ? intval($closeTimeSetting) : 5; // 默认为5分钟
        
        $time = time() - ($minutes * 60);

        // 查找需要关闭的订单
        $expiredOrders = Db::name('pay_order')
            ->where('state', 0)
            ->where('create_date', '<', $time)
            ->select()
            ->toArray();
        
        if (empty($expiredOrders)) {
            return $this->success(['count' => 0], '没有需要关闭的过期订单');
        }

        $closedCount = 0;
        $orderIdsToDelete = [];
        $orderPrimaryIdsToUpdate = [];

        foreach ($expiredOrders as $order) {
            $orderIdsToDelete[] = $order['order_id'];
            $orderPrimaryIdsToUpdate[] = $order['id'];
        }

        // 批量更新订单状态为已关闭
        if (!empty($orderPrimaryIdsToUpdate)) {
            $closedCount = Db::name('pay_order')
                ->where('id', 'in', $orderPrimaryIdsToUpdate)
                ->update(['state' => -1, 'close_date' => time()]);
        }

        // 批量删除 tmp_price 表中的记录
        if (!empty($orderIdsToDelete)) {
            Db::name('tmp_price')
                ->where('oid', 'in', $orderIdsToDelete)
                ->delete();
        }
            
        return $this->success(['count' => $closedCount], "成功关闭 {$closedCount} 条过期订单");
    }
    
    /**
     * 删除最后一天的订单
     * @return \think\Response
     */
    public function deleteLast()
    {
        $time = time() - 86400; // 一天前的订单
        $result = Db::name('pay_order')
            ->where('create_date', '<', $time)
            ->delete();
            
        return $this->success(['count' => $result], '删除历史订单成功');
    }
    
    /**
     * 获取订单状态文本
     * @param int $state 订单状态
     * @return string 状态文本
     */
    private function getOrderStateText($state)
    {
        switch ($state) {
            case -1:
                return '已关闭';
            case 0:
                return '未支付';
            case 1:
                return '已支付';
            default:
                return '未知状态';
        }
    }
    
    /**
     * 生成带签名的返回URL
     * @param string $id 订单ID
     * @return \think\Response
     */
    public function generateReturnUrl($id)
    {
        try {
            // 获取订单ID
            $orderId = $id;
            
            if (empty($orderId)) {
                return $this->error('订单号不能为空');
            }
            
            // 查询订单
            $order = Db::name('pay_order')->where('order_id', $orderId)->find();
            if (!$order) {
                return $this->error('订单不存在');
            }
            
            // 获取系统密钥
            $key = Db::name('setting')->where('vkey', 'key')->value('vvalue');
            if (empty($key)) {
                return $this->error('系统未配置密钥');
            }
            
            // 准备参数
            $payId = $order['pay_id'];
            $param = $order['param'] ?: '';
            $type = $order['type'];
            $price = number_format($order['price'], 2, '.', '');
            $reallyPrice = number_format($order['really_price'], 2, '.', '');
            
            // 计算签名
            $sign = md5($payId . $param . $type . $price . $reallyPrice . $key);
            
            // 构建参数字符串
            $params = "payId=" . urlencode($payId) . "&param=" . urlencode($param) . "&type=" . $type . "&price=" . $price . "&reallyPrice=" . $reallyPrice . "&sign=" . $sign;
            
            // 检查返回URL是否存在
            if (empty($order['return_url'])) {
                return $this->error('订单没有配置返回URL');
            }
            
            // 构建完整的返回URL
            $returnUrl = $order['return_url'];
            $separator = (strpos($returnUrl, '?') === false) ? '?' : '&';
            $fullReturnUrl = $returnUrl . $separator . $params;
            
            return $this->success([
                'returnUrl' => $fullReturnUrl,
                'sign' => $sign
            ]);
        } catch (\Exception $e) {
            // 记录错误日志
            error_log("生成返回URL异常: " . $e->getMessage() . "\n" . $e->getTraceAsString());
            return $this->error('服务器处理请求时发生错误');
        }
    }
} 