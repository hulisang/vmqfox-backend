<?php
namespace app\controller\api;

use think\Response;

class BaseController
{
    /**
     * 成功响应
     * @param mixed $data 响应数据
     * @param string $msg 响应消息
     * @param int $code 响应状态码
     * @return Response
     */
    protected function success($data = null, $msg = '成功', $code = 200)
    {
        return $this->response($data, $msg, $code);
    }
    
    /**
     * 错误响应
     * @param string $msg 错误消息
     * @param int $code 错误状态码
     * @param mixed $data 响应数据
     * @return Response
     */
    protected function error($msg = '请求失败', $code = 400, $data = null)
    {
        return $this->response($data, $msg, $code);
    }
    
    /**
     * 响应输出
     * @param mixed $data 响应数据
     * @param string $msg 响应消息
     * @param int $code 响应状态码
     * @return Response
     */
    protected function response($data, $msg, $code)
    {
        return Response::create([
            'code' => $code,
            'msg' => $msg,
            'data' => $data
        ], 'json');
    }
} 