<?php
/**
 * Created by PhpStorm.
 * User: vone
 * Date: 2019/4/16
 * Time: 22:13
 */

namespace app\service;

use Endroid\QrCode\QrCode;
use Endroid\QrCode\Writer\PngWriter;
use Endroid\QrCode\ErrorCorrectionLevel\ErrorCorrectionLevelHigh;
use Endroid\QrCode\Color\Color;
use Endroid\QrCode\Label\Label;
use Endroid\QrCode\Logo\Logo;

class QrcodeServer
{
    protected $_qr;
    protected $_encoding        = 'UTF-8';              // 编码类型
    protected $_size            = 180;                  // 二维码大小
    protected $_logo            = false;                // 是否需要带logo的二维码
    protected $_logo_url        = '';                   // logo图片路径
    protected $_logo_size       = 80;                   // logo大小
    protected $_title           = false;                // 是否需要二维码title
    protected $_title_content   = '';                   // title内容
    protected $_generate        = 'display';            // display-直接显示  writefile-写入文件
    protected $_file_name       = './static/qrcode';    // 写入文件路径
    const MARGIN           = 10;                        // 二维码内容相对于整张图片的外边距
    const WRITE_NAME       = 'png';                     // 写入文件的后缀名
    const FOREGROUND_COLOR = [0, 0, 0];                 // 前景色
    const BACKGROUND_COLOR = [255, 255, 255];           // 背景色

    public function __construct($config = []) {
        isset($config['generate'])      &&  $this->_generate        = $config['generate'];
        isset($config['encoding'])      &&  $this->_encoding        = $config['encoding'];
        isset($config['size'])          &&  $this->_size            = $config['size'];
        isset($config['logo'])          &&  $this->_logo            = $config['logo'];
        isset($config['logo_url'])      &&  $this->_logo_url        = $config['logo_url'];
        isset($config['logo_size'])     &&  $this->_logo_size       = $config['logo_size'];
        isset($config['title'])         &&  $this->_title           = $config['title'];
        isset($config['title_content']) &&  $this->_title_content   = $config['title_content'];
        isset($config['file_name'])     &&  $this->_file_name       = $config['file_name'];
    }

    /**
     * 生成二维码
     * @param $content //需要写入的内容
     * @return array | page input
     */
    public function createQrcode($content) {
        // 检查是否已经缓存过该二维码
        $cacheKey = md5($content);
        $cacheFile = runtime_path() . 'qrcode/' . $cacheKey . '.png';
        
        // 确保缓存目录存在
        if (!is_dir(runtime_path() . 'qrcode/')) {
            mkdir(runtime_path() . 'qrcode/', 0777, true);
        }
        
        // 如果缓存存在，直接返回
        if (file_exists($cacheFile)) {
            return 'data:image/png;base64,' . base64_encode(file_get_contents($cacheFile));
        }
        
        // 创建QR码
        $qrCode = QrCode::create($content)
            ->setSize($this->_size)
            ->setMargin(self::MARGIN)
            ->setErrorCorrectionLevel(new ErrorCorrectionLevelHigh())
            ->setForegroundColor(new Color(self::FOREGROUND_COLOR[0], self::FOREGROUND_COLOR[1], self::FOREGROUND_COLOR[2]))
            ->setBackgroundColor(new Color(self::BACKGROUND_COLOR[0], self::BACKGROUND_COLOR[1], self::BACKGROUND_COLOR[2]));

        // 是否需要logo
        if ($this->_logo && $this->_logo_url) {
            $logo = Logo::create($this->_logo_url)
                ->setResizeToWidth($this->_logo_size)
                ->setResizeToHeight($this->_logo_size);
            $qrCode->setLogo($logo);
        }

        // 是否需要title
        if ($this->_title && $this->_title_content) {
            $label = Label::create($this->_title_content)
                ->setTextColor(new Color(self::FOREGROUND_COLOR[0], self::FOREGROUND_COLOR[1], self::FOREGROUND_COLOR[2]));
            $qrCode->setLabel($label);
        }

        // 创建写入器
        $writer = new PngWriter();

        if ($this->_generate == 'display') {
            // 展示二维码
            $result = $writer->write($qrCode);
            
            // 保存到缓存
            file_put_contents($cacheFile, $result->getString());
            
            return $result->getDataUri();
        } else if ($this->_generate == 'writefile') {
            // 写入文件
            $file_name = $this->_file_name;
            return $this->generateImg($file_name, $writer, $qrCode);
        } else {
            return ['success' => false, 'message' => 'the generate type not found', 'data' => ''];
        }
    }

    /**
     * 生成文件
     * @param $file_name //目录文件 例: /tmp
     * @param $writer
     * @param $qrCode
     * @return array
     */
    public function generateImg($file_name, $writer, $qrCode) {
        $file_path = $file_name . DIRECTORY_SEPARATOR . uniqid() . '.' . self::WRITE_NAME;

        if (!file_exists($file_name)) {
            mkdir($file_name, 0777, true);
        }

        try {
            $result = $writer->write($qrCode);
            $result->saveToFile($file_path);
            
            $data = [
                'url' => $file_path,
                'ext' => self::WRITE_NAME,
            ];
            return ['success' => true, 'message' => 'write qrimg success', 'data' => $data];
        } catch (\Exception $e) {
            return ['success' => false, 'message' => $e->getMessage(), 'data' => ''];
        }
    }

    /**
     * 兼容旧方法名
     */
    public function createServer($content) {
        return $this->createQrcode($content);
    }
} 