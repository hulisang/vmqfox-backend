<?php
declare (strict_types = 1);

namespace app;

use think\Service;
use think\facade\Event;
use think\facade\Db;

/**
 * 应用服务类
 */
class AppService extends Service
{
    public function register()
    {
        // 服务注册
    }

    public function boot()
    {
        // 服务启动
        
        // 注册定时任务，每分钟自动关闭过期订单
        Event::listen('HttpRun', function () {
            // 随机执行，避免每次请求都执行
            if (mt_rand(1, 10) === 1) {
                $this->closeExpiredOrders();
            }
        });
    }
    
    /**
     * 关闭过期订单
     */
    private function closeExpiredOrders()
    {
        // 记录开始时间
        $startTime = microtime(true);
        
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
            return;
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
        
        // 记录执行时间和结果
        $endTime = microtime(true);
        $executionTime = round(($endTime - $startTime) * 1000, 2);
        
        if ($closedCount > 0) {
            error_log("自动关闭过期订单：成功关闭 {$closedCount} 条订单，耗时 {$executionTime}ms");
        }
    }
} 