# 概念

## DataSet

DataSet 表示与工作负载相关的数据集合，可以包括多个数据项。资源描述的示例如下:
```yaml
apiVersion: data.kuda.io/v1alpha1
kind: DataSet
metadata:
  name: dataset-nginx
spec:
  template:
    dataItems:
      - name: nginx
        namespace: kuda-io
        remotePath: /kuda/conf/test.conf
        localPath: /tmp/test.conf
        version: "v0.0.1"
        dataSourceType: hdfs
        lifecycle:
          postDownload:
            exec:
              command: ["/bin/bash", "-c", "cp /kuda/data/tmp/test.conf /etc/nginx/conf.d/test.conf && nginx -s reload"]
    dataSources:
      hdfs:
        addresses: ["hdfs-service.kuda-system:8020"]
        userName: root
  workloadSelector:
    app: "nginx"
```

关键字段的含义为:

* template: 用于描述应用数据的具体内容，包括数据项列表、数据源和自定义生命周期，具体含义参考 [Data](#Data) 部分。
* workloadSelector: 描述目标工作负载的标签，该 DataSet 将在匹配标签的所有实例上生效。

## Data

Data 表示工作负载具体实例对应的数据集合，除了描述当前实例所需的数据项之外，还维护了各项数据的具体状态。
该资源对象由 Kuda 组件根据 DataSet 和工作负载信息动态生成，不需要用户显示描述。示例如下:
```yaml
apiVersion: data.kuda.io/v1alpha1
kind: Data
metadata:
  labels:
    kuda.io/dataset: dataset-nginx
    kuda.io/pod: nginx-deployment-79767f796-dt2qf
  name: dataset-nginx-79767f796-dt2qf
  namespace: default
spec:
  dataItems:
    - dataSourceType: hdfs
      lifecycle:
        postDownload:
          exec:
            command:
              - /bin/bash
              - -c
              - cp /kuda/data/tmp/test.conf /etc/nginx/conf.d/test.conf && nginx -s reload
      localPath: /tmp/test.conf
      name: nginx
      namespace: kuda-io
      remotePath: /kuda/conf/test.conf
      version: v0.0.1
  dataSources:
    hdfs:
      addresses:
        - hdfs-service.kuda-system:8020
      userName: root
status:
  dataItems: 1
  dataItemsStatus:
    - name: nginx
      namespace: kuda-io
      phase: success
      startTime: "2021-11-08T07:13:34Z"
      version: v0.0.1
  downloading: 0
  failed: 0
  ready: 1/1
  success: 1
  waiting: 0

```

关键字段的含义为:

* dataItems: 数据项列表，每个数据项包含名称、命名空间、版本等属性信息
    * name: 数据项名称，必须确保相同 namespace 下数据名称是唯一的
    * namespace: 数据项命名空间，用于多个数据项的分组管理
    * remotePath: 数据项在存储端的路径，也就是数据源端的存储路径
    * localPath: 数据项下载后的本地路径，即业务容器中看到的数据路径(注意: 最终路径需要加上前缀`/kuda/data`)
    * version: 数据版本，数据变更后应填写不同的版本号，方便数据的版本管理和回滚等操作
    * dataSourceType: 数据源类型，该类型必须在 dataSources 中存在
    * lifecycle: 支持在数据下载前和下载后添加自定义操作，包括 exec 和 httpGet 两种方式
* dataSources: 定义不同的数据源，目前支持 hdfs 和 alluxio 两种

