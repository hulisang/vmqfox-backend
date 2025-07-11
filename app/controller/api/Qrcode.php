<?php
namespace app\controller\api;

use think\facade\Db;
use think\facade\Request;
use app\service\QrcodeServer;
use Zxing\QrReader;

class Qrcode extends BaseController
{
    /**
     * 获取二维码列表
     * @return \think\Response
     */
    public function list()
    {
        $page = Request::param('page', 1);
        $limit = Request::param('limit', 10);
        $type = Request::param('type');
        
        $where = [];
        if ($type !== null && $type !== '') {
            $where[] = ['type', '=', $type];
        }
        
        $count = Db::name('pay_qrcode')->where($where)->count();
        $list = Db::name('pay_qrcode')
            ->where($where)
            ->page($page, $limit)
            ->order('id desc')
            ->select()
            ->toArray();
            
        foreach ($list as &$item) {
            $item['type_text'] = $item['type'] == 1 ? '微信' : '支付宝';
            $item['state_text'] = $item['state'] == 0 ? '正常' : '禁用';
        }
        
        return $this->success([
            'total' => $count,
            'items' => $list
        ]);
    }
    
    /**
     * 添加二维码
     * @return \think\Response
     */
    public function add()
    {
        $type = Request::param('type');
        $payUrl = Request::param('pay_url');
        $price = Request::param('price');
        
        if (empty($type) || ($type != 1 && $type != 2)) {
            return $this->error('支付类型错误');
        }
        
        if (empty($payUrl)) {
            return $this->error('收款码不能为空');
        }
        
        $data = [
            'type' => $type,
            'pay_url' => $payUrl,
            'price' => $price ?: 0,
            'state' => 0
        ];
        
        $result = Db::name('pay_qrcode')->insert($data);
        if (!$result) {
            return $this->error('添加二维码失败');
        }
        
        return $this->success(null, '添加二维码成功');
    }
    
    /**
     * 删除二维码
     * @param int $id 二维码ID
     * @return \think\Response
     */
    public function delete($id)
    {
        $result = Db::name('pay_qrcode')->where('id', $id)->delete();
        if (!$result) {
            return $this->error('删除二维码失败');
        }
        
        return $this->success(null, '删除二维码成功');
    }
    
    /**
     * 解析二维码
     * @return \think\Response
     */
    public function parse()
    {
        // 使用 Request::file() 来获取上传的文件
        $file = Request::file('file');
        
        // 判断文件是否存在
        if (empty($file)) {
            return $this->error('二维码数据不能为空，请选择文件上传');
        }
        
        // 解析二维码
        try {
            // 直接使用 Zxing\QrReader 解析上传的文件
            $qrcode = new QrReader($file->getRealPath());
            $result = $qrcode->text();
            
            return $this->success([
                'url' => $result
            ]);
        } catch (\Throwable $e) { // 使用 Throwable 捕获所有类型的错误
            // 返回更详细的错误信息以供调试
            $errorMessage = sprintf(
                '解析二维码失败: %s (文件: %s, 行号: %d)',
                $e->getMessage(),
                $e->getFile(),
                $e->getLine()
            );
            return $this->error($errorMessage);
        }
    }
    
    /**
     * 获取微信二维码列表
     * @return \think\Response
     */
    public function wechat()
    {
        $page = Request::param('page', 1);
        $limit = Request::param('limit', 10);
        
        $count = Db::name('pay_qrcode')->where('type', 1)->count();
        $list = Db::name('pay_qrcode')
            ->where('type', 1)
            ->page($page, $limit)
            ->order('id desc')
            ->select()
            ->toArray();
            
        foreach ($list as &$item) {
            $item['state_text'] = $item['state'] == 0 ? '正常' : '禁用';
        }
        
        return $this->success([
            'total' => $count,
            'items' => $list
        ]);
    }
    
    /**
     * 获取支付宝二维码列表
     * @return \think\Response
     */
    public function alipay()
    {
        $page = Request::param('page', 1);
        $limit = Request::param('limit', 10);
        
        $count = Db::name('pay_qrcode')->where('type', 2)->count();
        $list = Db::name('pay_qrcode')
            ->where('type', 2)
            ->page($page, $limit)
            ->order('id desc')
            ->select()
            ->toArray();
            
        foreach ($list as &$item) {
            $item['state_text'] = $item['state'] == 0 ? '正常' : '禁用';
        }
        
        return $this->success([
            'total' => $count,
            'items' => $list
        ]);
    }
    
    /**
     * 添加微信二维码
     * @return \think\Response
     */
    public function addWechat()
    {
        $payUrl = Request::param('pay_url');
        $price = Request::param('price');
        
        if (empty($payUrl)) {
            return $this->error('收款码不能为空');
        }
        
        $data = [
            'type' => 1,
            'pay_url' => $payUrl,
            'price' => $price ?: 0,
            'state' => 0
        ];
        
        $result = Db::name('pay_qrcode')->insert($data);
        if (!$result) {
            return $this->error('添加微信二维码失败');
        }
        
        return $this->success(null, '添加微信二维码成功');
    }
    
    /**
     * 添加支付宝二维码
     * @return \think\Response
     */
    public function addAlipay()
    {
        $payUrl = Request::param('pay_url');
        $price = Request::param('price');
        
        if (empty($payUrl)) {
            return $this->error('收款码不能为空');
        }
        
        $data = [
            'type' => 2,
            'pay_url' => $payUrl,
            'price' => $price ?: 0,
            'state' => 0
        ];
        
        $result = Db::name('pay_qrcode')->insert($data);
        if (!$result) {
            return $this->error('添加支付宝二维码失败');
        }
        
        return $this->success(null, '添加支付宝二维码成功');
    }
    
    /**
     * 删除微信二维码
     * @param int $id 二维码ID
     * @return \think\Response
     */
    public function deleteWechat($id)
    {
        $qrcode = Db::name('pay_qrcode')->where('id', $id)->find();
        if (!$qrcode) {
            return $this->error('二维码不存在');
        }
        
        if ($qrcode['type'] != 1) {
            return $this->error('该二维码不是微信二维码');
        }
        
        $result = Db::name('pay_qrcode')->where('id', $id)->delete();
        if (!$result) {
            return $this->error('删除微信二维码失败');
        }
        
        return $this->success(null, '删除微信二维码成功');
    }
    
    /**
     * 删除支付宝二维码
     * @param int $id 二维码ID
     * @return \think\Response
     */
    public function deleteAlipay($id)
    {
        $qrcode = Db::name('pay_qrcode')->where('id', $id)->find();
        if (!$qrcode) {
            return $this->error('二维码不存在');
        }
        
        if ($qrcode['type'] != 2) {
            return $this->error('该二维码不是支付宝二维码');
        }
        
        $result = Db::name('pay_qrcode')->where('id', $id)->delete();
        if (!$result) {
            return $this->error('删除支付宝二维码失败');
        }
        
        return $this->success(null, '删除支付宝二维码成功');
    }
    
    /**
     * 设置二维码绑定状态
     * @param int $id 二维码ID
     * @return \think\Response
     */
    public function bind($id)
    {
        $state = Request::param('state');
        
        if ($state === null) {
            return $this->error('状态参数不能为空');
        }
        
        $qrcode = Db::name('pay_qrcode')->where('id', $id)->find();
        if (!$qrcode) {
            return $this->error('二维码不存在');
        }
        
        $result = Db::name('pay_qrcode')->where('id', $id)->update([
            'state' => $state
        ]);
        
        if (!$result) {
            return $this->error('设置二维码状态失败');
        }
        
        return $this->success(null, '设置二维码状态成功');
    }
    
    /**
     * 生成二维码
     * @return \think\Response
     */
    public function generate()
    {
        // 获取URL参数
        $url = Request::param('url');
        
        if (empty($url)) {
            return $this->error('URL参数不能为空');
        }
        
        try {
            // 创建QrcodeServer实例
            $qrcodeServer = new QrcodeServer();
            
            // 生成二维码
            $qrcode = $qrcodeServer->createQrcode($url);
            
            // 从data URI中提取图像数据
            if (preg_match('/^data:image\/(\w+);base64,(.+)$/', $qrcode, $matches)) {
                $imageType = $matches[1];
                $imageData = base64_decode($matches[2]);
                
                // 设置正确的Content-Type
                return response($imageData)->header(['Content-Type' => 'image/' . $imageType]);
            }
            
            // 如果是其他格式的返回值，直接返回成功
            return $this->success(['qrcode' => $qrcode]);
        } catch (\Exception $e) {
            return $this->error('生成二维码失败: ' . $e->getMessage());
        }
    }
} 