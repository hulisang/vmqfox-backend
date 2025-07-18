<?php

// +----------------------------------------------------------------------
// | ThinkPHP [ WE CAN DO IT JUST THINK ]
// +----------------------------------------------------------------------
// | Copyright (c) 2006~2025 http://thinkphp.cn All rights reserved.
// +----------------------------------------------------------------------
// | Licensed ( http://www.apache.org/licenses/LICENSE-2.0 )
// +----------------------------------------------------------------------
// | Author: liu21st <liu21st@gmail.com>
// +----------------------------------------------------------------------
declare (strict_types = 1);

namespace think;

use BackedEnum;
use Closure;
use think\contract\Enumable;
use think\exception\ValidateException;
use think\helper\Arr;
use think\helper\Str;
use think\validate\ValidateRule;
use think\validate\ValidateRuleSet;
use UnitEnum;

/**
 * 数据验证类
 * @package think
 */
class Validate
{
    /**
     * 自定义验证类型
     * @var array
     */
    protected $type = [];

    /**
     * 验证类型别名
     * @var array
     */
    protected $map = [
        '>' => 'gt', '>=' => 'egt', '<' => 'lt', '<=' => 'elt', '=' => 'eq', 'same' => 'eq', '<>' => 'neq',
    ];

    /**
     * 当前验证规则
     * @var array
     */
    protected $rule = [];

    /**
     * 当前验证规则集
     * @var array
     */
    protected $ruleSet = [];

    /**
     * 当前验证规则组
     * @var array
     */
    protected $group = [];

    /**
     * 验证规则别名
     * @var array
     */
    protected $alias = [];

    /**
     * 验证提示信息
     * @var array
     */
    protected $message = [];

    /**
     * 验证字段描述
     * @var array
     */
    protected $field = [];

    /**
     * 默认规则提示
     * @var array
     */
    protected $typeMsg = [
        'require'     => ':attribute require',
        'must'        => ':attribute must',
        'number'      => ':attribute must be numeric',
        'integer'     => ':attribute must be integer',
        'float'       => ':attribute must be float',
        'string'      => ':attribute must be string',
        'boolean'     => ':attribute must be bool',
        'enum'        => ':attribute must be :rule enum',
        'email'       => ':attribute not a valid email address',
        'mobile'      => ':attribute not a valid mobile',
        'array'       => ':attribute must be a array',
        'accepted'    => ':attribute must be yes,on,true or 1',
        'acceptedIf'  => ':attribute must be yes,on,true or 1',
        'declined'    => ':attribute must be no,off,false or 0',
        'declinedIf'  => ':attribute must be no,off,false or 0',
        'date'        => ':attribute not a valid datetime',
        'file'        => ':attribute not a valid file',
        'image'       => ':attribute not a valid image',
        'alpha'       => ':attribute must be alpha',
        'alphaNum'    => ':attribute must be alpha-numeric',
        'alphaDash'   => ':attribute must be alpha-numeric, dash, underscore',
        'activeUrl'   => ':attribute not a valid domain or ip',
        'chs'         => ':attribute must be chinese',
        'chsAlpha'    => ':attribute must be chinese or alpha',
        'chsAlphaNum' => ':attribute must be chinese,alpha-numeric',
        'chsDash'     => ':attribute must be chinese,alpha-numeric,underscore, dash',
        'url'         => ':attribute not a valid url',
        'ip'          => ':attribute not a valid ip',
        'dateFormat'  => ':attribute must be dateFormat of :rule',
        'in'          => ':attribute must be in :rule',
        'notIn'       => ':attribute be notin :rule',
        'between'     => ':attribute must between :1 - :2',
        'notBetween'  => ':attribute not between :1 - :2',
        'length'      => 'size of :attribute must be :rule',
        'max'         => 'max size of :attribute must be :rule',
        'min'         => 'min size of :attribute must be :rule',
        'after'       => ':attribute cannot be less than :rule',
        'before'      => ':attribute cannot exceed :rule',
        'expire'      => ':attribute not within :rule',
        'allowIp'     => 'access IP is not allowed',
        'denyIp'      => 'access IP denied',
        'confirm'     => ':attribute out of accord with :2',
        'different'   => ':attribute cannot be same with :2',
        'egt'         => ':attribute must greater than or equal :rule',
        'gt'          => ':attribute must greater than :rule',
        'elt'         => ':attribute must less than or equal :rule',
        'lt'          => ':attribute must less than :rule',
        'eq'          => ':attribute must equal :rule',
        'neq'         => ':attribute must not be equal to :rule',
        'unique'      => ':attribute has exists',
        'regex'       => ':attribute not conform to the rules',
        'method'      => 'invalid Request method',
        'token'       => 'invalid token',
        'fileSize'    => 'filesize not match',
        'fileExt'     => 'extensions to upload is not allowed',
        'fileMime'    => 'mimetype to upload is not allowed',
        'startWith'   => ':attribute must start with :rule',
        'endWith'     => ':attribute must end with :rule',
        'contain'     => ':attribute must contain :rule',
        'multipleOf'  => ':attribute must multiple :rule',
    ];

    protected $typeMsgZh = [
        'require'     => ':attribute不能为空',
        'must'        => ':attribute必须',
        'number'      => ':attribute必须是数字',
        'integer'     => ':attribute必须是整数',
        'float'       => ':attribute必须是浮点数',
        'string'      => ':attribute必须是字符串',
        'enum'        => ':attribute必须是有效的 :rule 枚举',
        'startWith'   => ':attribute必须以 :rule 开头',
        'endWith'     => ':attribute必须以 :rule 结尾',
        'contain'     => ':attribute必须包含 :rule',
        'boolean'     => ':attribute必须是布尔值',
        'email'       => ':attribute格式不符',
        'mobile'      => ':attribute格式不符',
        'array'       => ':attribute必须是数组',
        'accepted'    => ':attribute必须是yes、on、true或者1',
        'acceptedIf'  => ':attribute必须是yes、on、true或者1',
        'declined'    => ':attribute必须是no、off、false或者0',
        'declinedIf'  => ':attribute必须是no、off、false或者0',
        'date'        => ':attribute不是一个有效的日期或时间格式',
        'file'        => ':attribute不是有效的上传文件',
        'image'       => ':attribute不是有效的图像文件',
        'alpha'       => ':attribute只能是字母',
        'alphaNum'    => ':attribute只能是字母和数字',
        'alphaDash'   => ':attribute只能是字母、数字和下划线_及破折号-',
        'activeUrl'   => ':attribute不是有效的域名或者IP',
        'chs'         => ':attribute只能是汉字',
        'chsAlpha'    => ':attribute只能是汉字、字母',
        'chsAlphaNum' => ':attribute只能是汉字、字母和数字',
        'chsDash'     => ':attribute只能是汉字、字母、数字和下划线_及破折号-',
        'url'         => ':attribute不是有效的URL地址',
        'ip'          => ':attribute不是有效的IP地址',
        'dateFormat'  => ':attribute必须使用日期格式 :rule',
        'in'          => ':attribute必须在 :rule 范围内',
        'notIn'       => ':attribute不能在 :rule 范围内',
        'between'     => ':attribute只能在 :1 - :2 之间',
        'notBetween'  => ':attribute不能在 :1 - :2 之间',
        'length'      => ':attribute长度不符合要求 :rule',
        'max'         => ':attribute长度不能超过 :rule',
        'min'         => ':attribute长度不能小于 :rule',
        'after'       => ':attribute日期不能小于 :rule',
        'before'      => ':attribute日期不能超过 :rule',
        'expire'      => '不在有效期内 :rule',
        'allowIp'     => '不允许的IP访问',
        'denyIp'      => '禁止的IP访问',
        'confirm'     => ':attribute和确认字段:2不一致',
        'different'   => ':attribute和比较字段:2不能相同',
        'egt'         => ':attribute必须大于等于 :rule',
        'gt'          => ':attribute必须大于 :rule',
        'elt'         => ':attribute必须小于等于 :rule',
        'lt'          => ':attribute必须小于 :rule',
        'eq'          => ':attribute必须等于 :rule',
        'neq'         => ':attribute不能等于 :rule',
        'unique'      => ':attribute已存在',
        'regex'       => ':attribute不符合指定规则',
        'multipleOf'  => ':attribute必须是 :rule 的倍数',
        'fileSize'    => '文件大小不符',
        'fileExt'     => '文件后缀不允许',
        'fileMime'    => '文件类型不允许',
        'method'      => '无效的请求类型',
        'token'       => '令牌数据无效',
    ];

    /**
     * 当前验证场景
     * @var string
     */
    protected $currentScene;

    /**
     * 内置正则验证规则
     * @var array
     */
    protected $defaultRegex = [
        'alpha'       => '/^[A-Za-z]+$/',
        'alphaNum'    => '/^[A-Za-z0-9]+$/',
        'alphaDash'   => '/^[A-Za-z0-9\-\_]+$/',
        'chs'         => '/^[\p{Han}]+$/u',
        'chsAlpha'    => '/^[\p{Han}a-zA-Z]+$/u',
        'chsAlphaNum' => '/^[\p{Han}a-zA-Z0-9]+$/u',
        'chsDash'     => '/^[\p{Han}a-zA-Z0-9\_\-]+$/u',
        'mobile'      => '/^1[3-9]\d{9}$/',
        'idCard'      => '/(^[1-9]\d{5}(18|19|([23]\d))\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$)|(^[1-9]\d{5}\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}$)/',
        'zip'         => '/\d{6}/',
    ];

    /**
     * Filter_var 规则
     * @var array
     */
    protected $filter = [
        'email'   => FILTER_VALIDATE_EMAIL,
        'ip'      => [FILTER_VALIDATE_IP, FILTER_FLAG_IPV4 | FILTER_FLAG_IPV6],
        'integer' => FILTER_VALIDATE_INT,
        'url'     => FILTER_VALIDATE_URL,
        'macAddr' => FILTER_VALIDATE_MAC,
        'float'   => FILTER_VALIDATE_FLOAT,
    ];

    /**
     * 验证场景定义
     * @var array
     */
    protected $scene = [];

    /**
     * 验证失败错误信息
     * @var string|array
     */
    protected $error = [];

    /**
     * 是否批量验证
     * @var bool
     */
    protected $batch = false;

    /**
     * 验证失败是否抛出异常
     * @var bool
     */
    protected $failException = false;

    /**
     * 必须验证的规则
     * @var array
     */
    protected $must = [];

    /**
     * 场景需要验证的规则
     * @var array
     */
    protected $only = [];

    /**
     * 场景需要移除的验证规则
     * @var array
     */
    protected $remove = [];

    /**
     * 场景需要追加的验证规则
     * @var array
     */
    protected $append = [];

    /**
     * 场景需要覆盖的验证规则
     * @var array
     */
    protected $replace = [];

    /**
     * 验证正则定义
     * @var array
     */
    protected $regex = [];

    /**
     * Db对象
     * @var Db
     */
    protected $db;

    /**
     * 语言对象
     * @var Lang
     */
    protected $lang;

    /**
     * 请求对象
     * @var Request
     */
    protected $request;

    /**
     * @var Closure[]
     */
    protected static $maker = [];

    /**
     * 构造方法
     */
    public function __construct()
    {
        if (!empty(static::$maker)) {
            foreach (static::$maker as $maker) {
                call_user_func($maker, $this);
            }
        }
    }

    /**
     * 设置服务注入
     * @param Closure $maker
     * @return void
     */
    public static function maker(Closure $maker)
    {
        static::$maker[] = $maker;
    }

    /**
     * 设置Lang对象
     * @param Lang $lang Lang对象
     * @return void
     */
    public function setLang($lang)
    {
        $this->lang = $lang;
    }

    /**
     * 使用中文提示
     * @return $this
     */
    public function useZh()
    {
        $this->typeMsg = array_merge($this->typeMsg, $this->typeMsgZh);
        return $this;
    }

    /**
     * 设置Db对象
     * @param Db $db Db对象
     * @return void
     */
    public function setDb($db)
    {
        $this->db = $db;
    }

    /**
     * 设置Request对象
     * @param Request $request Request对象
     * @return void
     */
    public function setRequest($request)
    {
        $this->request = $request;
    }

    /**
     * 添加字段验证规则
     * @param string|array $name 字段名称或者规则数组
     * @param mixed        $rule 验证规则或者字段描述信息
     * @param string|array $msg  错误信息
     * @return $this
     */
    public function rule(string | array $name, $rule = '', string | array $msg = [])
    {
        if (is_array($name)) {
            $this->rule = $name + $this->rule;
            if (is_array($rule)) {
                $this->field = array_merge($this->field, $rule);
            }
            if (!empty($msg) && is_array($msg)) {
                $this->message($msg);
            }
        } else {
            $this->rule[$name] = $rule;
            if (!empty($msg)) {
                $this->message[$name] = $msg;
            }
        }

        return $this;
    }

    /**
     * 获取所有验证规则
     * @return array
     */
    public function getRule(): array
    {
        return $this->rule;
    }

    /**
     * 添加验证规则别名
     * @param string|array $name 验证规则别名或者别名数组
     * @param mixed        $rule 验证规则
     * @return $this
     */
    public function alias(string | array $name, $rule = '')
    {
        if (is_array($name)) {
            $this->alias = $name + $this->alias;
            if (is_array($rule)) {
                $this->field = array_merge($this->field, $rule);
            }
        } else {
            $this->alias[$name] = $rule;
        }

        return $this;
    }

    /**
     * 添加验证规则组 支持闭包或数组 每个分组的验证彼此独立
     * group('group',function($validate) {
     *      return $validate->rule('name','rule...');
     * })
     * @param string        $name  分组名
     * @param array|Closure $rules 验证规则
     * @return $this
     */
    public function group(string $name, array | Closure $rules)
    {
        $this->group[$name] = $rules;

        return $this;
    }

    /**
     * 添加验证规则集 支持闭包或数组
     * @param string        $name  分组名
     * @param array|Closure $rules 验证规则
     * @param array         $msg   错误信息
     * @return ValidateRuleSet
     */
    public function ruleSet(string $name, array | Closure $rules, array $msg = [])
    {
        $this->rule[$name . '.*'] = ValidateRuleSet::rules($rules, $msg);

        return $this;
    }

    /**
     * 注册验证（类型）规则
     * @param string   $type     验证规则类型
     * @param callable $callback callback方法(或闭包)
     * @param string   $message  验证失败提示信息
     * @return $this
     */
    public function extend(string $type, callable $callback, ?string $message = null)
    {
        $this->type[$type] = $callback;

        if ($message) {
            $this->typeMsg[$type] = $message;
        }

        return $this;
    }

    /**
     * 设置验证规则的默认提示信息
     * @param string|array $type 验证规则类型名称或者数组
     * @param string       $msg  验证提示信息
     * @return void
     */
    public function setTypeMsg(string | array $type, ?string $msg = null): void
    {
        if (is_array($type)) {
            $this->typeMsg = array_merge($this->typeMsg, $type);
        } else {
            $this->typeMsg[$type] = $msg;
        }
    }

    /**
     * 设置提示信息
     * @param array $message 错误信息
     * @return $this
     */
    public function message(array $message)
    {
        $this->message = array_merge($this->message, $message);

        return $this;
    }

    /**
     * 获取提示信息
     * @return array
     */
    public function getMessage(): array
    {
        return $this->message;
    }

    /**
     * 设置验证场景或直接指定需要验证的字段
     * @param string|array $name 场景名
     * @return $this
     */
    public function scene(string | array $name)
    {
        if (is_array($name)) {
            $this->only = $name;
        } else {
            // 设置当前场景
            $this->currentScene = $name;
        }

        return $this;
    }

    /**
     * 判断是否存在某个验证场景
     * @param string $name 场景名
     * @return bool
     */
    public function hasScene(string $name): bool
    {
        return isset($this->scene[$name]) || method_exists($this, 'scene' . Str::studly($name));
    }

    /**
     * 设置批量验证
     * @param bool $batch 是否批量验证
     * @return $this
     */
    public function batch(bool $batch = true)
    {
        $this->batch = $batch;

        return $this;
    }

    /**
     * 设置验证失败后是否抛出异常
     * @param bool $fail 是否抛出异常
     * @return $this
     */
    public function failException(bool $fail = true)
    {
        $this->failException = $fail;

        return $this;
    }

    /**
     * 指定需要验证的字段列表
     * @param array $fields 字段名
     * @return $this
     */
    public function only(array $fields)
    {
        $this->only = $fields;

        return $this;
    }

    /**
     * 指定需要覆盖的字段验证规则
     * @param string $field 字段名
     * @param mixed  $rules 验证规则
     * @return $this
     */
    public function replace(string $field, $rules)
    {
        $this->replace[$field] = $rules;

        return $this;
    }

    /**
     * 移除某个字段的验证规则
     * @param string|array $field 字段名
     * @param mixed        $rule  验证规则 true 移除所有规则
     * @return $this
     */
    public function remove(string | array $field, $rule = null)
    {
        if (is_array($field)) {
            foreach ($field as $key => $rule) {
                if (is_int($key)) {
                    $this->remove($rule);
                } else {
                    $this->remove($key, $rule);
                }
            }
        } else {
            if (is_string($rule)) {
                $rule = explode('|', $rule);
            }

            $this->remove[$field] = $rule;
        }

        return $this;
    }

    /**
     * 追加某个字段的验证规则
     * @param string|array $field 字段名
     * @param mixed        $rule  验证规则
     * @return $this
     */
    public function append(string | array $field, $rule = null)
    {
        if (is_array($field)) {
            foreach ($field as $key => $rule) {
                $this->append($key, $rule);
            }
        } else {
            if (is_string($rule)) {
                $rule = explode('|', $rule);
            }

            $this->append[$field] = $rule;
        }

        return $this;
    }

    /**
     * 返回定义的验证规则
     * @return mixed
     */
    protected function rules()
    {
        return $this->rule;
    }

    /**
     * 获取当前的验证规则
     * @return array
     */
    public function getRules(): array
    {
        return $this->rule;
    }

    /**
     * 数据自动验证
     * @param array $data  数据
     * @param array|string $rules 验证规则
     * @return bool
     */
    public function check(array $data, array | string $rules = []): bool
    {
        $this->error = [];

        if (empty($rules)) {
            // 读取验证规则
            $rules = $this->rules();
        } elseif (is_string($rules)) {
            $rules = $this->getGroupRules($rules);
            // 分组独立检测
            if ($rules instanceof Closure) {
                return $rules(new self())
                    ->alias($this->alias)
                    ->batch($this->batch)
                    ->failException($this->failException)
                    ->check($data);
            }
        }

        if ($rules instanceof Validate) {
            $rules = $rules->getRules();
        }

        if ($this->currentScene) {
            $this->getScene($this->currentScene);
        }

        foreach ($this->append as $key => $rule) {
            if (!isset($rules[$key])) {
                $rules[$key] = $rule;
                unset($this->append[$key]);
            }
        }

        foreach ($rules as $key => $rule) {
            if (str_contains($key, '|')) {
                // 字段|描述 用于指定属性名称
                [$key, $title] = explode('|', $key);
            } else {
                $title = $this->field[$key] ?? $key;
            }

            // 场景检测
            if (!empty($this->only) && (!in_array($key, $this->only) && !array_key_exists($key, $this->only))) {
                continue;
            }

            // 数据验证
            $result = $this->checkItems($key, $rule, $data, $title);
            if (false === $result) {
                return false;
            }
        }

        if (!empty($this->error)) {
            if ($this->failException) {
                throw new ValidateException($this->error);
            }
            return false;
        }

        return true;
    }

    /**
     * 返回经过验证的数据，只包含验证规则中的数据
     * @param array $data
     * @param array|string $rules
     * @return array
     */
    public function checked(array $data, array | string $rules = []): array
    {
        $checkRes = $this->check($data, $rules);

        if (!$checkRes) {
            throw new ValidateException($this->error);
        }

        $results      = [];
        $missingValue = Str::random(10);

        foreach (array_keys($this->getRules()) as $key) {
            if (str_contains($key, '|')) {
                [$key] = explode('|', $key);
            }
            $value = data_get($data, $key, $missingValue);

            if ($value !== $missingValue) {
                Arr::set($results, $key, $value);
            }
        }

        return $results;
    }

    /**
     * 验证字段规则
     * @param string $key   字段名
     * @param mixed  $rule  验证规则
     * @param array  $data  数据
     * @param string $title 字段描述
     * @return bool
     */
    protected function checkItems($key, $rule, $data, $title): bool
    {
        if ($rule instanceof ValidateRuleSet) {
            // 验证集
            $values = $this->getDataSet($data, $key);
            if (empty($values)) {
                return true;
            }
            $items = $rule->getRules();
            if ($items instanceof Closure) {
                // 获取验证集的规则
                $items = $items(new self())->getRule();
            }
            // 验证集的错误信息
            foreach ($rule->getMessage() as $name => $message) {
                $this->message[$key . '.' . $name] = $message;
            }
        } else {
            $items = [$rule];
        }

        foreach ($items as $k => $item) {
            $name = is_string($k) ? $key . '.' . $k : $key;
            if (str_contains($name, '|')) {
                // 字段|描述 用于指定属性名称
                [$name, $title] = explode('|', $name);
            }

            $values = $this->getDataSet($data, $name);
            if (empty($values)) {
                $values[$name] = null;
            }

            foreach ($values as $value) {
                $result = $this->checkItem($name, $value, $item, $data, $title);
                if (true !== $result) {
                    // 验证失败 记录错误信息
                    if (false === $result) {
                        $result = $this->getRuleMsg($name, $title, '', $item);
                    }

                    $this->error[$name] = $result;
                    if ($this->batch) {
                        // 批量验证
                    } elseif ($this->failException) {
                        throw new ValidateException($result, $name);
                    } else {
                        return false;
                    }
                }
            }
        }
        return true;
    }

    /**
     * 根据验证规则验证数据
     * @param mixed $value 字段值
     * @param mixed $rules 验证规则
     * @return bool
     */
    public function checkRule($value, $rules): bool
    {
        if ($rules instanceof Closure) {
            $result = call_user_func_array($rules, [$value]);
            return is_bool($result) ? $result : false;
        }

        if ($rules instanceof ValidateRule) {
            $rules = $rules->getRule();
        } elseif (is_string($rules)) {
            $rules = explode('|', $rules);
        }

        foreach ($rules as $key => $rule) {
            if ($rule instanceof Closure) {
                $result = call_user_func_array($rule, [$value]);
            } elseif (is_subclass_of($rule, UnitEnum::class) || is_subclass_of($rule, Enumable::class)) {
                $result = $this->enum($value, $rule);
            } else {
                // 判断验证类型
                [$type, $rule, $callback] = $this->getValidateType($key, $rule);

                $result = call_user_func_array($callback, [$value, $rule]);
            }

            if (true !== $result) {
                return false;
            }
        }

        return true;
    }

    /**
     * 验证单个字段规则
     * @param string $field 字段名
     * @param mixed  $value 字段值
     * @param mixed  $rules 验证规则
     * @param array  $data  数据
     * @param string $title 字段描述
     * @param array  $msg   提示信息
     * @return mixed
     */
    protected function checkItem(string $field, $value, $rules, $data, string $title = '', array $msg = []): mixed
    {
        if ($rules instanceof ValidateRuleSet) {
            return $this->checkItems($field, $rules, $data, $title);
        }

        if ($rules instanceof Closure) {
            return call_user_func_array($rules, [$value, $data]);
        }

        if ($rules instanceof ValidateRule) {
            $title = $rules->getTitle() ?: $title;
            $msg   = $rules->getMsg();
            $rules = $rules->getRule();
        } elseif (is_string($rules)) {
            // 检查验证规则别名
            // 'alias' => require|in:a,b,c|... 或者 ['require','in'=>'a,b,c',...]
            $rules = $this->alias[$rules] ?? $rules;
        }

        if (isset($this->remove[$field]) && true === $this->remove[$field] && empty($this->append[$field])) {
            // 字段已经移除 无需验证
            return true;
        }

        if (isset($this->replace[$field])) {
            $rules = $this->replace[$field];
        } elseif (isset($this->only[$field])) {
            $rules = $this->only[$field];
        }

        // 统一解析为数组规则验证 require|in:a,b,c|... 或者 ['require','in'=>'a,b,c',...]
        if (is_string($rules)) {
            $rules = explode('|', $rules);
        }

        if (isset($this->append[$field])) {
            // 追加额外的验证规则
            $rules = array_unique(array_merge($rules, $this->append[$field]), SORT_REGULAR);
            unset($this->append[$field]);
        }

        if (empty($rules)) {
            return true;
        }

        $i = 0;
        foreach ($rules as $key => $rule) {
            if ($rule instanceof Closure) {
                $result = call_user_func_array($rule, [$value, $data]);
                $type   = is_numeric($key) ? '' : $key;
            } elseif (is_subclass_of($rule, UnitEnum::class) || is_subclass_of($rule, Enumable::class)) {
                $result = $this->enum($value, $rule);
                $type   = is_numeric($key) ? '' : $key;
            } else {
                // 判断验证类型
                [$type, $rule, $callback] = $this->getValidateType($key, $rule);

                if (isset($this->append[$field]) && in_array($type, $this->append[$field])) {
                } elseif (isset($this->remove[$field]) && in_array($type, $this->remove[$field])) {
                    // 规则已经移除
                    $i++;
                    continue;
                }

                if ('must' == $type || str_starts_with($type, 'require') || in_array($field, $this->must) || (!is_null($value) && '' !== $value)) {
                    $result = call_user_func_array($callback, [$value, $rule, $data, $field, $title]);
                } else {
                    $result = true;
                }
            }

            if (false === $result) {
                // 验证失败 返回错误信息
                if (!empty($msg[$i])) {
                    $message = $msg[$i];
                    if (is_string($message) && str_starts_with($message, '{%')) {
                        $message = $this->lang->get(substr($message, 2, -1));
                    }
                } else {
                    $message = $this->getRuleMsg($field, $title, $type, $rule);
                }

                return $message;
            } elseif (true !== $result) {
                // 返回自定义错误信息
                return $this->parseUserErrorMessage($result, $title, $rule);
            }
            $i++;
        }

        return $result ?? true;
    }

    protected function parseUserErrorMessage($message, $title, $rule)
    {
        if (is_string($message) && str_contains($message, ':')) {
            $message = str_replace(':attribute', $title, $message);

            if (str_contains($message, ':rule') && is_scalar($rule)) {
                $message = str_replace(':rule', (string) $rule, $message);
            }
        }

        return $message;
    }

    /**
     * 获取当前验证类型及规则
     * @param mixed $key
     * @param mixed $rule
     * @return array
     */
    protected function getValidateType($key, $rule): array
    {
        // 判断验证类型
        $hasParam = true;
        if (!is_numeric($key)) {
            $type = $key;
        } elseif (str_contains($rule, ':')) {
            [$type, $rule] = explode(':', $rule, 2);
        } else {
            $type     = $rule;
            $hasParam = false;
        }

        // 验证类型别名
        $type = $this->map[$type] ?? $type;

        if (isset($this->type[$type])) {
            // 自定义验证
            $call = $this->type[$type];
        } else {
            $method = Str::camel($type);
            if (method_exists($this, $method)) {
                $call = [$this, $method];
                $rule = $hasParam ? $rule : '';
            } else {
                $call = [$this, 'is'];
            }
        }

        return [$type, $rule, $call];
    }

    /**
     * 验证是否和某个字段的值一致
     * @param mixed  $value 字段值
     * @param mixed  $rule  验证规则
     * @param array  $data  数据
     * @param string $field 字段名
     * @return bool
     */
    public function confirm($value, $rule, array $data = [], string $field = ''): bool
    {
        if ('' == $rule) {
            if (str_contains($field, '_confirm')) {
                $rule = strstr($field, '_confirm', true);
            } else {
                $rule = $field . '_confirm';
            }
        }

        return $this->getDataValue($data, $rule) === $value;
    }

    /**
     * 验证是否和某个字段的值是否不同
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @param array $data  数据
     * @return bool
     */
    public function different($value, $rule, array $data = []): bool
    {
        return $this->getDataValue($data, $rule) != $value;
    }

    /**
     * 验证是否大于等于某个值
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @param array $data  数据
     * @return bool
     */
    public function egt($value, $rule, array $data = []): bool
    {
        return $value >= $this->getDataValue($data, $rule);
    }

    /**
     * 验证是否大于某个值
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @param array $data  数据
     * @return bool
     */
    public function gt($value, $rule, array $data = []): bool
    {
        return $value > $this->getDataValue($data, $rule);
    }

    /**
     * 验证是否小于等于某个值
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @param array $data  数据
     * @return bool
     */
    public function elt($value, $rule, array $data = []): bool
    {
        return $value <= $this->getDataValue($data, $rule);
    }

    /**
     * 验证是否小于某个值
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @param array $data  数据
     * @return bool
     */
    public function lt($value, $rule, array $data = []): bool
    {
        return $value < $this->getDataValue($data, $rule);
    }

    /**
     * 验证是否等于某个值
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function eq($value, $rule): bool
    {
        return $value == $rule;
    }

    /**
     * 验证是否不等于某个值
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function neq($value, $rule): bool
    {
        return $value != $rule;
    }

    /**
     * 必须验证
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function must($value, $rule = null): bool
    {
        return !empty($value) || '0' == $value;
    }

    /**
     * 验证字段值是否为有效格式
     * @param mixed  $value 字段值
     * @param string $rule  验证规则
     * @param array  $data  数据
     * @return bool
     */
    public function is($value, string $rule, array $data = []): bool
    {
        $call = function ($value, $rule) {
            if (function_exists('ctype_' . $rule)) {
                // ctype验证规则
                $ctypeFun = 'ctype_' . $rule;
                $result   = $ctypeFun((string) $value);
            } elseif (isset($this->filter[$rule])) {
                // Filter_var验证规则
                $result = $this->filter($value, $this->filter[$rule]);
            } else {
                // 正则验证
                $result = $this->regex($value, $rule);
            }
            return $result;
        };

        return match (Str::camel($rule)) {
            'require'         => !empty($value) || '0' == $value,
            'accepted'        => in_array($value, ['1', 'on', 'yes', 'true', 1, true], true),
            'declined'        => in_array($value, ['0', 'off', 'no', 'false', 0, false], true),
            'boolean', 'bool' => in_array($value, [true, false, 'true', 'false', 0, 1, '0', '1'], true),
            'date'            => false !== strtotime($value),
            'activeUrl'       => checkdnsrr($value),
            'number'          => is_numeric($value),
            'alphaNum'        => ctype_alnum((string)$value),
            'array'           => is_array($value),
            'integer', 'int'  => is_numeric($value) && is_int((int)$value),
            'float'           => is_numeric($value) && is_float((float)$value),
            'string'          => is_string($value),
            'file'            => $value instanceof File,
            'image'           => $value instanceof File && in_array($this->getImageType($value->getRealPath()), [1, 2, 3, 6, 9, 10, 11, 14, 15, 17, 18]),
            'token'           => $this->token($value, '__token__', $data),
            default           => $call($value, $rule),
        };
    }

    // 判断图像类型
    protected function getImageType($image)
    {
        if (function_exists('exif_imagetype')) {
            return exif_imagetype($image);
        }

        try {
            $info = getimagesize($image);
            return $info ? $info[2] : false;
        } catch (\Exception $e) {
            return false;
        }
    }

    /**
     * 验证表单令牌
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @param array $data  数据
     * @return bool
     */
    public function token($value, string $rule, array $data): bool
    {
        if ($this->request) {
            $rule = !empty($rule) ? $rule : '__token__';
            return $this->request->checkToken($rule, $data);
        }
        return true;
    }

    /**
     * 验证是否为合格的域名或者IP 支持A，MX，NS，SOA，PTR，CNAME，AAAA，A6， SRV，NAPTR，TXT 或者 ANY类型
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function activeUrl(string $value, string $rule = 'MX'): bool
    {
        if (!in_array($rule, ['A', 'MX', 'NS', 'SOA', 'PTR', 'CNAME', 'AAAA', 'A6', 'SRV', 'NAPTR', 'TXT', 'ANY'])) {
            $rule = 'MX';
        }

        return checkdnsrr($value, $rule);
    }

    /**
     * 验证是否有效IP
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则 ipv4 ipv6
     * @return bool
     */
    public function ip($value, string $rule = 'ipv4'): bool
    {
        if (!in_array($rule, ['ipv4', 'ipv6'])) {
            $rule = 'ipv4';
        }

        return $this->filter($value, [FILTER_VALIDATE_IP, 'ipv6' == $rule ? FILTER_FLAG_IPV6 : FILTER_FLAG_IPV4]);
    }

    /**
     * 检测是否以某个字符串开头
     * @param mixed $value 字段值
     * @param string $rule  验证规则
     * @return bool
     */
    public function startWith($value, string $rule): bool
    {
        return is_string($value) && str_starts_with($value, $rule);
    }

    /**
     * 检测是否以某个字符串结尾
     * @param mixed $value 字段值
     * @param string $rule  验证规则
     * @return bool
     */
    public function endWith($value, string $rule): bool
    {
        return is_string($value) && str_ends_with($value, $rule);
    }

    /**
     * 检测是否以包含某个字符串
     * @param mixed $value 字段值
     * @param string $rule  验证规则
     * @return bool
     */
    public function contain($value, string $rule): bool
    {
        return is_string($value) && str_contains($value, $rule);
    }

    /**
     * 检测上传文件后缀
     * @param File         $file
     * @param array|string $ext 允许后缀
     * @return bool
     */
    protected function checkExt(File $file, $ext): bool
    {
        if (is_string($ext)) {
            $ext = explode(',', $ext);
        }

        return in_array(strtolower($file->extension()), $ext);
    }

    /**
     * 检测上传文件大小
     * @param File    $file
     * @param integer $size 最大大小
     * @return bool
     */
    protected function checkSize(File $file, $size): bool
    {
        return $file->getSize() <= (int) $size;
    }

    /**
     * 检测上传文件类型
     * @param File         $file
     * @param array|string $mime 允许类型
     * @return bool
     */
    protected function checkMime(File $file, $mime): bool
    {
        if (is_string($mime)) {
            $mime = explode(',', $mime);
        }

        return in_array(strtolower($file->getMime()), $mime);
    }

    /**
     * 验证上传文件后缀
     * @param mixed $file 上传文件
     * @param mixed $rule 验证规则
     * @return bool
     */
    public function fileExt($file, $rule): bool
    {
        if (is_array($file)) {
            foreach ($file as $item) {
                if (!($item instanceof File) || !$this->checkExt($item, $rule)) {
                    return false;
                }
            }
            return true;
        } elseif ($file instanceof File) {
            return $this->checkExt($file, $rule);
        }

        return false;
    }

    /**
     * 验证上传文件类型
     * @param mixed $file 上传文件
     * @param mixed $rule 验证规则
     * @return bool
     */
    public function fileMime($file, $rule): bool
    {
        if (is_array($file)) {
            foreach ($file as $item) {
                if (!($item instanceof File) || !$this->checkMime($item, $rule)) {
                    return false;
                }
            }
            return true;
        } elseif ($file instanceof File) {
            return $this->checkMime($file, $rule);
        }

        return false;
    }

    /**
     * 验证上传文件大小
     * @param mixed $file 上传文件
     * @param mixed $rule 验证规则
     * @return bool
     */
    public function fileSize($file, $rule): bool
    {
        if (is_array($file)) {
            foreach ($file as $item) {
                if (!($item instanceof File) || !$this->checkSize($item, $rule)) {
                    return false;
                }
            }
            return true;
        } elseif ($file instanceof File) {
            return $this->checkSize($file, $rule);
        }

        return false;
    }

    /**
     * 验证图片的宽高及类型
     * @param mixed $file 上传文件
     * @param mixed $rule 验证规则
     * @return bool
     */
    public function image($file, $rule): bool
    {
        if (is_array($file)) {
            foreach ($file as $item) {
                if (!($item instanceof File) || !$this->checkImage($item, $rule)) {
                    return false;
                }
            }
            return true;
        } elseif ($file instanceof File) {
            return $this->checkImage($file, $rule);
        }

        return false;        
    }

    /**
     * 验证图片的宽高及类型
     * @param mixed $file 上传文件
     * @param mixed $rule 验证规则
     * @return bool
     */
    public function checkImage($file, $rule): bool
    {
        if ($rule) {
            $rule = explode(',', $rule);

            if (0 === strpos($rule[0], 'type=')) {
                $type      = array_shift($rule);
                $imageType = substr($type, 5);
            } elseif (isset($rule[2])) {
                $imageType = strtolower($rule[2]);
            }

            [$width, $height, $type] = getimagesize($file->getRealPath());
            if (isset($imageType)) {
                if ('jpg' == $imageType) {
                    $imageType = 'jpeg';
                }

                if (image_type_to_extension($type, false) != strtolower($imageType)) {
                    return false;
                }
            }

            if (count($rule) > 1) {
                [$w, $h] = $rule;
                return $w == $width && $h == $height;
            }
            return true;
        }

        return in_array($this->getImageType($file->getRealPath()), [1, 2, 3, 6, 9, 10, 11, 14, 15, 17, 18]);
    }

    /**
     * 验证时间和日期是否符合指定格式
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function dateFormat($value, $rule): bool
    {
        $info = date_parse_from_format($rule, $value);
        if (strlen((string) $info['year']) != 4 && strpos($rule, 'Y') !== false) {
            return false;
        }
        return 0 == $info['warning_count'] && 0 == $info['error_count'];
    }

    /**
     * 验证是否唯一
     * @param mixed  $value 字段值
     * @param mixed  $rule  验证规则 格式：数据表,字段名,排除ID,主键名
     * @param array  $data  数据
     * @param string $field 验证字段名
     * @return bool
     */
    public function unique($value, $rule, array $data = [], string $field = ''): bool
    {
        if (!$this->db) {
            return true;
        }

        if (is_string($rule)) {
            $rule = explode(',', $rule);
        }

        if (str_contains($rule[0], '\\')) {
            // 指定模型类
            $db = new $rule[0]();
        } else {
            $db = $this->db->name($rule[0]);
        }

        $key = $rule[1] ?? $field;
        $map = [];

        if (str_contains($key, '^')) {
            // 支持多个字段验证
            $fields = explode('^', $key);
            foreach ($fields as $key) {
                if (isset($data[$key])) {
                    $map[] = [$key, '=', $data[$key]];
                }
            }
        } elseif (strpos($key, '=')) {
            // 支持复杂验证
            parse_str($key, $array);
            foreach ($array as $k => $val) {
                $map[] = [$k, '=', $data[$k] ?? $val];
            }
        } elseif (isset($data[$field])) {
            $map[] = [$key, '=', $data[$field]];
        }

        $pk = !empty($rule[3]) ? $rule[3] : $db->getPk();

        if (is_string($pk)) {
            if (isset($rule[2])) {
                $map[] = [$pk, '<>', $rule[2]];
            } elseif (isset($data[$pk])) {
                $map[] = [$pk, '<>', $data[$pk]];
            }
        }

        if ($db->where($map)->field($pk)->find()) {
            return false;
        }

        return true;
    }

    /**
     * 使用filter_var方式验证
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function filter($value, $rule): bool
    {
        if (is_string($rule) && str_contains($rule, ',')) {
            [$rule, $param] = explode(',', $rule);
        } elseif (is_array($rule)) {
            $param = $rule[1] ?? 0;
            $rule  = $rule[0];
        } else {
            $param = 0;
        }

        return false !== filter_var($value, is_int($rule) ? $rule : filter_id($rule), $param);
    }

    /**
     * 验证某个字段等于某个值的时候必须
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @param array $data  数据
     * @return bool
     */
    public function requireIf($value, $rule, array $data = []): bool
    {
        [$field, $val] = is_string($rule) ? explode(',', $rule) : $rule;

        if ($this->getDataValue($data, $field) == $val) {
            return !empty($value) || '0' == $value;
        }

        return true;
    }

    /**
     * 通过回调方法验证某个字段是否必须
     * @param mixed        $value 字段值
     * @param string|array $rule  验证规则
     * @param array        $data  数据
     * @return bool
     */
    public function requireCallback($value, string | array $rule, array $data = []): bool
    {
        $callback = is_array($rule) ? $rule : [$this, $rule];
        $result   = call_user_func_array($callback, [$value, $data]);

        if ($result) {
            return !empty($value) || '0' == $value;
        }

        return true;
    }

    /**
     * 验证某个字段有值的情况下必须
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @param array $data  数据
     * @return bool
     */
    public function requireWith($value, $rule, array $data = []): bool
    {
        $val = $this->getDataValue($data, $rule);

        if (!empty($val)) {
            return !empty($value) || '0' == $value;
        }

        return true;
    }

    /**
     * 验证某个字段没有值的情况下必须
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @param array $data  数据
     * @return bool
     */
    public function requireWithout($value, $rule, array $data = []): bool
    {
        $val = $this->getDataValue($data, $rule);

        if (empty($val)) {
            return !empty($value) || '0' == $value;
        }

        return true;
    }

    /**
     * 验证是否为数组，支持检查键名
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function array($value, $rule): bool
    {
        if (!is_array($value)) {
            return false;
        }
        if ($rule) {
            $keys = is_string($rule) ? explode(',', $rule) : $rule;
            return empty(array_diff($keys, array_keys($value)));
        } else {
            return true;
        }
    }

    /**
     * 验证是否在范围内
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function in($value, $rule): bool
    {
        return in_array($value, is_array($rule) ? $rule : explode(',', $rule));
    }

    /**
     * 验证是否为枚举
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function enum($value, $rule): bool
    {
        if (is_subclass_of($rule, BackedEnum::class)) {
            $values = array_map(fn($case) => $case->value, $rule::cases());
        } elseif (is_subclass_of($rule, UnitEnum::class)) {
            $values = array_map(fn($case) => $case->name, $rule::cases());
        } elseif (is_subclass_of($rule, Enumable::class)) {
            $values = $rule::values();
        } else {
            $reflect = new \ReflectionClass($rule);
            $values  = $reflect->getConstants();
        }

        return in_array($value, $values ?? []);
    }

    /**
     * 验证是否不在某个范围
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function notIn($value, $rule): bool
    {
        return !in_array($value, is_array($rule) ? $rule : explode(',', $rule));
    }

    /**
     * between验证数据
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function between($value, $rule): bool
    {
        [$min, $max] = is_string($rule) ? explode(',', $rule) : $rule;

        return $value >= $min && $value <= $max;
    }

    /**
     * 使用notbetween验证数据
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function notBetween($value, $rule): bool
    {
        [$min, $max] = is_string($rule) ? explode(',', $rule) : $rule;

        return $value < $min || $value > $max;
    }

    /**
     * 验证数据长度
     * @param mixed                  $value 字段值
     * @param string|array|int|float $rule  验证规则
     * @return bool
     */
    public function length($value, $rule): bool
    {
        if (is_array($value)) {
            $length = count($value);
        } elseif ($value instanceof File) {
            $length = $value->getSize();
        } else {
            $length = mb_strlen((string) $value);
        }

        if (is_array($rule)) {
            // 长度区间
            return $length >= $rule[0] && $length <= $rule[1];
        } elseif (is_string($rule) && str_contains($rule, ',')) {
            // 长度区间
            [$min, $max] = explode(',', $rule);
            return $length >= $min && $length <= $max;
        }

        // 指定长度
        return $length == $rule;
    }

    /**
     * 验证数据最大长度
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function max($value, $rule): bool
    {
        if (is_array($value)) {
            $length = count($value);
        } elseif ($value instanceof File) {
            $length = $value->getSize();
        } else {
            $length = mb_strlen((string) $value);
        }

        return $length <= $rule;
    }

    /**
     * 验证数据最小长度
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function min($value, $rule): bool
    {
        if (is_array($value)) {
            $length = count($value);
        } elseif ($value instanceof File) {
            $length = $value->getSize();
        } else {
            $length = mb_strlen((string) $value);
        }

        return $length >= $rule;
    }

    /**
     * 验证日期
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @param array $data  数据
     * @return bool
     */
    public function after($value, $rule, array $data = []): bool
    {
        return strtotime($value) >= strtotime($rule);
    }

    /**
     * 验证日期
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @param array $data  数据
     * @return bool
     */
    public function before($value, $rule, array $data = []): bool
    {
        return strtotime($value) <= strtotime($rule);
    }

    /**
     * 验证日期
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @param array $data  数据
     * @return bool
     */
    public function afterWith($value, $rule, array $data = []): bool
    {
        $rule = $this->getDataValue($data, $rule);
        return !is_null($rule) && strtotime($value) >= strtotime($rule);
    }

    /**
     * 验证日期
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @param array $data  数据
     * @return bool
     */
    public function beforeWith($value, $rule, array $data = []): bool
    {
        $rule = $this->getDataValue($data, $rule);
        return !is_null($rule) && strtotime($value) <= strtotime($rule);
    }

    /**
     * 验证有效期
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function expire($value, $rule): bool
    {
        [$start, $end] = is_string($rule) ? explode(',', $rule) : $rule;

        if (!is_numeric($start)) {
            $start = strtotime($start);
        }

        if (!is_numeric($end)) {
            $end = strtotime($end);
        }

        return time() >= $start && time() <= $end;
    }

    /**
     * 验证IP许可
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function allowIp($value, $rule): bool
    {
        return in_array($value, is_array($rule) ? $rule : explode(',', $rule));
    }

    /**
     * 验证IP禁用
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则
     * @return bool
     */
    public function denyIp($value, $rule): bool
    {
        return !in_array($value, is_array($rule) ? $rule : explode(',', $rule));
    }

    /**
     * 验证某个字段等于指定的值，则验证中的字段必须为 yes、on、1 或 true
     * @param mixed $value 字段值
     * @param mixed $rule 验证规则
     * @param array $data 数据
     * @return bool
     */
    public function acceptedIf($value, $rule, array $data = []): bool
    {
        [$field, $val] = is_string($rule) ? explode(',', $rule) : $rule;

        if ($this->getDataValue($data, $field) == $val) {
            return in_array($value, ['1', 'on', 'yes', 'true', 1, true], true);
        }

        return true;
    }

    /**
     * 验证某个字段等于指定的值，则验证中的字段必须为 no、off、0 或 false
     * @param mixed $value 字段值
     * @param mixed $rule 验证规则
     * @param array $data 数据
     * @return bool
     */
    public function declinedIf($value, $rule, array $data = []): bool
    {
        [$field, $val] = is_string($rule) ? explode(',', $rule) : $rule;

        if ($this->getDataValue($data, $field) == $val) {
            return in_array($value, ['0', 'off', 'no', 'false', 0, false], true);
        }

        return true;
    }

    /**
     * 验证某个字段必须是指定值的倍数
     * @param mixed $value 字段值
     * * @param mixed $rule 验证规则
     * @return bool
     */
    public function multipleOf($value, $rule): bool
    {
        if ('0' == $rule || $value < $rule) {
            return false;
        }

        return $value % $rule === 0;
    }

    /**
     * 使用正则验证数据
     * @param mixed $value 字段值
     * @param mixed $rule  验证规则 正则规则或者预定义正则名
     * @return bool
     */
    public function regex($value, $rule): bool
    {
        $rule = $this->regex[$rule] ?? $this->getDefaultRegexRule($rule);

        if (is_string($rule) && !str_starts_with($rule, '/') && !preg_match('/\/[imsU]{0,4}$/', $rule)) {
            // 不是正则表达式则两端补上/
            $rule = '/^' . $rule . '$/';
        }

        return is_scalar($value) && 1 === preg_match($rule, (string) $value);
    }

    /**
     * 获取内置正则验证规则
     * @param string $rule  验证规则 正则规则或者预定义正则名
     * @return string
     */
    protected function getDefaultRegexRule(string $rule): string
    {
        $name = Str::camel($rule);
        if (isset($this->defaultRegex[$name])) {
            $rule = $this->defaultRegex[$name];
        }
        return $rule;
    }

    /**
     * 获取错误信息
     * @param bool  $withKey 是否包含字段信息
     * @return array|string
     */
    public function getError(bool $withKey = false)
    {
        if ($withKey || count($this->error) > 1) {
            // 批量验证
            return $this->error;
        }
        return empty($this->error) ? '' : array_values($this->error)[0];
    }

    /**
     * 获取数据集合
     * @param array  $data 数据
     * @param string $key  数据标识 支持二维
     * @return array
     */
    protected function getDataSet(array $data, $key): array
    {
        if (is_string($key) && str_contains($key, '*')) {
            if (substr_count($key, '*') > 1) {
                [$key1, $key2] = explode('.*.', $key, 2);

                $array = $this->getDataValue($data, $key1);
                $data  = is_array($array) ? $this->getDataSet($data, $key1 . '.*')[0] : [];

                return $this->getDataSet($data, $key2);
            }

            if (str_ends_with($key, '*')) {
                // user.id.*
                [$key] = explode('.*', $key);
                $value = $this->getRecursiveData($data, $key) ?: [];
                return is_array($value) ? $value : [$value];
            }
            // user.*.id
            [$key, $column] = explode('.*.', $key);

            $value = $this->getRecursiveData($data, $key);
            if (!is_array($value)) {
                $value = [];
            }

            return array_map(function ($item) use ($column) {
                return $item[$column] ?? null;
            }, $value);
        }

        $value = $this->getDataValue($data, $key);
        return is_null($value) ? [] : [$value];
    }

    /**
     * 获取数据值
     * @param array  $data 数据
     * @param string $key  数据标识 支持二维
     * @return mixed
     */
    protected function getDataValue(array $data, $key)
    {
        if (is_numeric($key)) {
            $value = $key;
        } elseif (is_string($key) && str_contains($key, '.')) {
            // 支持多维数组验证
            $value = $this->getRecursiveData($data, $key);
        } else {
            $value = $data[$key] ?? null;
        }

        return $value;
    }

    /**
     * 获取数据值
     * @param array  $data 数据
     * @param string $key  数据标识 支持二维
     * @return mixed
     */
    protected function getRecursiveData(array $data, string $key)
    {
        $keys = explode('.', $key);
        foreach ($keys as $key) {
            if (!isset($data[$key])) {
                $value = null;
                break;
            }
            $value = $data = $data[$key];
        }
        return $value;
    }

    /**
     * 获取验证规则的错误提示信息
     * @param string $attribute 字段英文名
     * @param string $title     字段描述名
     * @param string $type      验证规则名称
     * @param mixed  $rule      验证规则数据
     * @return string|array
     */
    protected function getRuleMsg(string $attribute, string $title, string $type, $rule)
    {
        if (isset($this->message[$attribute . '.' . $type])) {
            $msg = $this->message[$attribute . '.' . $type];
        } elseif (isset($this->message[$attribute][$type])) {
            $msg = $this->message[$attribute][$type];
        } elseif (isset($this->message[$attribute])) {
            $msg = $this->message[$attribute];
        } elseif (isset($this->typeMsg[$type])) {
            $msg = $this->typeMsg[$type];
        } elseif (str_starts_with($type, 'require')) {
            $msg = $this->typeMsg['require'];
        } else {
            $msg = $title . ($this->lang ? $this->lang->get('not conform to the rules') : '规则不符');
        }

        if (is_array($msg)) {
            return $this->errorMsgIsArray($msg, $rule, $title);
        }

        return $this->parseErrorMsg($msg, $rule, $title);
    }

    /**
     * 获取验证规则的错误提示信息
     * @param string $msg   错误信息
     * @param mixed  $rule  验证规则数据
     * @param string $title 字段描述名
     * @return string|array
     */
    protected function parseErrorMsg(string $msg, $rule, string $title)
    {
        if ($this->lang) {
            if (str_starts_with($msg, '{%')) {
                $msg = $this->lang->get(substr($msg, 2, -1));
            } elseif ($this->lang->has($msg)) {
                $msg = $this->lang->get($msg);
            }
        }

        if (is_array($msg)) {
            return $this->errorMsgIsArray($msg, $rule, $title);
        }

        // rule若是数组则转为字符串
        if (is_array($rule)) {
            $rule = implode(',', $rule);
        }

        if (is_scalar($rule) && str_contains($msg, ':')) {
            // 变量替换
            if (is_string($rule) && str_contains($rule, ',')) {
                $array = array_pad(explode(',', $rule), 3, '');
            } else {
                $array = array_pad([], 3, '');
            }

            $msg = str_replace(
                [':attribute', ':1', ':2', ':3'],
                [$title, $array[0], $array[1], $array[2]],
                $msg,
            );

            if (str_contains($msg, ':rule')) {
                $msg = str_replace(':rule', (string) $rule, $msg);
            }
        }

        return $msg;
    }

    /**
     * 错误信息数组处理
     * @param array $msg   错误信息
     * @param mixed  $rule  验证规则数据
     * @param string $title 字段描述名
     * @return array
     */
    protected function errorMsgIsArray(array $msg, $rule, string $title)
    {
        foreach ($msg as $key => $val) {
            if (is_string($val)) {
                $msg[$key] = $this->parseErrorMsg($val, $rule, $title);
            }
        }
        return $msg;
    }

    /**
     * 获取数据验证的场景
     * @param string $scene 验证场景
     * @return void
     */
    protected function getScene(string $scene): void
    {
        $method = 'scene' . Str::studly($scene);
        if (method_exists($this, $method)) {
            call_user_func([$this, $method]);
        } elseif (isset($this->scene[$scene])) {
            // 如果设置了验证适用场景
            $this->only = $this->scene[$scene];
        }
    }

    /**
     * 获取验证分组
     * @param string $group 分组名
     * @return mixed
     */
    protected function getGroupRules(string $group)
    {
        $method = 'rules' . Str::studly($group);
        if (method_exists($this, $method)) {
            $validate = call_user_func_array([$this, $method], [new self()]);
            return $validate->alias($this->alias)
                ->batch($this->batch)
                ->failException($this->failException);
        }
        return $this->group[$group] ?? [];
    }

    /**
     * 动态方法 直接调用is方法进行验证
     * @param string $method 方法名
     * @param array  $args   调用参数
     * @return bool
     */
    public function __call($method, $args)
    {
        if ('is' == strtolower(substr($method, 0, 2))) {
            $method = substr($method, 2);
        }

        array_push($args, lcfirst($method));

        return call_user_func_array([$this, 'is'], $args);
    }
}
