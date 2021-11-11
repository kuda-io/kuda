# 快速开始

这里使用 tensorflow serving 应用作为示例，演示模型下发并生效的过程。

## 准备工作

1、 按照 [安装手册](./install.md) 的说明安装平台组件和 HDFS 附加组件，并确保各组件正常工作

2、 准备模型文件 [half_plus_two](https://github.com/tensorflow/serving/tree/master/tensorflow_serving/servables/tensorflow/testdata/saved_model_half_plus_two_2_versions) 并上传到 HDFS:
```shell
# 下载 serving 项目并进入模型数据的目录
$ git clone https://github.com/tensorflow/serving
$ cd serving/tensorflow_serving/servables/tensorflow/testdata

# 设置 hdfs 实例的名称
$ export HDFS_POD=$(kubectl get pod -n kuda-system -l app=hdfs -o jsonpath={.items..metadata.name})

# 将本地模型文件拷贝到容器中
$ kubectl cp saved_model_half_plus_two_2_versions -n kuda-system $HDFS_POD:/opt/hadoop

# 在 hdfs 服务端创建目录
$ kubectl exec -n kuda-system $HDFS_POD -- hadoop fs -mkdir -p /kuda/models

# 将模型数据上传到 hdfs 服务端
$ kubectl exec -n kuda-system $HDFS_POD -- hadoop fs -put /opt/hadoop/saved_model_half_plus_two_2_versions /kuda/models/saved_model_half_plus_two
```

3、检查模型文件是否上传成功
```shell
$ kubectl exec -n kuda-system $HDFS_POD -- hadoop fs -ls /kuda/models/saved_model_half_plus_two
Found 2 items
drwxr-xr-x   - root supergroup          0 2021-10-29 08:39 /kuda/models/saved_model_half_plus_two/00000123
drwxr-xr-x   - root supergroup          0 2021-10-29 08:39 /kuda/models/saved_model_half_plus_two/00000124
```
   
## 操作步骤

1、 提交 DataSet 内容
```yaml
$ cat <<EOF | kubectl apply -f -
apiVersion: data.kuda.io/v1alpha1
kind: DataSet
metadata:
  name: dataset-serving
spec:
  template:
    dataItems:
      - name: half_plus_two
        namespace: kuda-io
        remotePath: /kuda/models/saved_model_half_plus_two/00000123
        localPath: /models/half_plus_two/00000123
        version: "00000123"
        dataSourceType: hdfs
    dataSources:
      hdfs:
        addresses: ["hdfs-service.kuda-system:8020"]
        userName: root
  workloadSelector:
    app: "serving"
EOF
```

2、创建 Tensorflow Serving 应用
```yaml
$ cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: serving-deployment
  labels:
    app: serving
spec:
  replicas: 1
  selector:
    matchLabels:
      app: serving
  template:
    metadata:
      labels:
        app: serving
    spec:
      containers:
        - name: serving
          image: tensorflow/serving
          ports:
            - containerPort: 8501
          env:
            - name: MODEL_NAME
              value: half_plus_two
            - name: MODEL_BASE_PATH
              value: /kuda/data/models
EOF
```

3、检查服务是否成功启动

```shell
$ kubectl get pod
NAME                                 READY   STATUS        RESTARTS   AGE
serving-deployment-b84855677-4l4h8   2/2     Running       0          28s
```
该实例已经自动注入 sidecar 容器，并成功启动。

4、检查数据下发状态 

```shell
$ kubectl get dataset
NAME              DATAITEMS   READY   AGE
dataset-serving   1           1/1     37s

$ kubectl get data
NAME                              READY   AGE
dataset-serving-cb7b558cf-6sn59   1/1     26s
```

可以看到，模型数据已经成功下发。

5、验证 serving 服务

执行下面的命令验证 serving 服务是否正常工作:
```shell
$ export SERVING_POD_IP=$(kubectl get pod -l app=serving -o jsonpath={.items..status.podIP})
$ kubectl run debug -i --rm --quiet=true --restart=Never --image=curlimages/curl -- curl -d '{"instances": [1.0, 2.0, 5.0]}' -X POST -s  http://$SERVING_POD_IP:8501/v1/models/half_plus_two:predict
{
    "predictions": [2.5, 3.0, 4.5
    ]
}
```
或者直接访问 version 特定的接口:
```shell
$ kubectl run debug -i --rm --quiet=true --restart=Never --image=curlimages/curl -- curl -d '{"instances": [1.0, 2.0, 5.0]}' -X POST -s  http://$SERVING_POD_IP:8501/v1/models/half_plus_two/versions/123:predict
{
    "predictions": [2.5, 3.0, 4.5
    ]
}
```

更进一步，您可以将 DataSet 中的模型版本改成 00000124 并提交更新，可以发现应用会加载新版本的模型并生效，模型更新结果符合预期。

## 清理

```shell
$ kubectl delete dataset dataset-serving
$ kubectl delete deployment serving-deployment
```