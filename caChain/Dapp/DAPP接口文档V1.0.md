[toc]

# 协作网络后端接口

## 通用

每个请求都遵循如下定义，除非特殊说明：

- 请求形式：json
- 方法：post
- 返回形式：json，当http响应状态码为:200 时，返回结构如下：

```json
{
  "code": 0,
  // 0代表成功， 其余为错误  
  "msg": "success"
  // success代表成功，错误时为错误信息
  //...所需数据项
}
```

## /v1/get_voteList 获取投票列表

- 请求体body :

```json
{
  "pageNum": 1,
  // 页数  1+
  "pageSize": 1
  // 数量  1-100
}
```

- 返回结构体

```json
{
  "code": 0,
  // 0代表成功， 其余为错误  
  "msg": "success",
  // success代表成功，错误时为错误信息
  "totalCount": "16",
  //数据总条目
  "voteList": [
    {
      "id": 1,
      //主键,投票id
      "title": "圣诞节最受欢迎的礼物投票",
      //投票项目名称
      "state": "进行中",
      // 投票状态
      "picUrl": "dfaslgjkdsa"
      //投票图片链接
      "title": "圣诞节最受欢迎的礼物投票",
      //投票标题
     "start_time": "2022-9-22",
     //投票开始时间
    "end_time": "2022-10-22",
    //投票结束时间
    "rule": "单选",
    //投票规则
    "scenery": "上周的课堂后，孩子们都拿到一张圣诞心愿卡片",
    //投票场景
    },
    {
      "id": 1,
      //主键,投票id  
      "title": "圣诞节最受欢迎的礼物投票",
      // 投票项目名称
      "state": "未开始",
      // 投票状态
      "picUrl": "dfaslgjkdsa"
      //投票图片链接
    }
  ]
}
```

## /v1/get_voteDetail 获取投票项详情

- 请求体body :

```json
{
  "voteId": 1,
  //投票id
  "pageNum": 1,
  // 页数  1+
  "pageSize": 1
  // 数量  1-100
}
```

- 返回结构体

```json
{
  "code": 0,
  // 0代表成功， 其余为错误  
  "msg": "success",
  // success代表成功，错误时为错误信息
  "title": "圣诞节最受欢迎的礼物投票",
  //投票标题
  "start_time": "2022-9-22",
  //投票开始时间
  "end_time": "2022-10-22",
  //投票结束时间
  "rule": "单选",
  //投票规则
  "scenery": "上周的课堂后，孩子们都拿到一张圣诞心愿卡片",
  //投票场景
  "voteDetail": [
    {
      "id": 1,
      //主键,投票项id
      "voteId": 1,
      //投票id
      "number": 100,
      // 当前投票项投票数量
      "sum_Number": 2000,
      // 投票总数
      "description": "这个礼物太可爱了，快来给我们投票吧"，
      //投票项说明
      "picItemUrl": "dfaslgjkdsa"
      //投票项图片链接
    },
    {
      "id": 2,
      //主键,投票项id
      "voteId": 1,
      //投票id
      "number": 100,
      // 当前投票项投票数量
      "sumNumber": 2000,
      // 投票总数
      "description": "这个礼物太可爱了，快来给我们投票吧",
      //投票说明
      "picItemUrl": "dfaslgjkdsa"
      //投票项图片链接
    }
  ]
}
```

## /v1/upload_File 上传文件

- 请求体body :

```json
{
  "id": 1,
  //主键,投票项id
  "voteId": 1,
  //投票id
  "FileHeader": 
    {
      "Filename": "圣诞节",
      "Header":   "map[天:[56 78] 明:[12 34]]",
      "Size":     64,
      
      "content": "abcd",
      "tmpfile": "efghi"
    }
}
```

- 返回结构体

```json
{
  "code": 0,
  // 0代表成功， 其余为错误  
  "msg": "success",
  // success代表成功，错误时为错误信息
  "url": "https://docs.qq.com/doc/DR0x3UHVTa3dyREJQ?&u=1bd71332337042a4816974652a52546b"
  //文件链接
}
```