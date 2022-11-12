# 说明

本示例演示将 nginx 配置文件下发到目标实例并生效的过程。

## 准备工作

1、 按照 [安装手册](../../docs/install.md) 的说明安装平台组件和 HDFS 附加组件，并确保各组件正常工作

2、 准备 nginx 配置文件并上传到 HDFS:
```shell
# 将以下内容写入 test.conf 文件，该文件表示nginx开启8081端口
$ cat <<EOF > test.conf
server {
    listen       8081;
    server_name  localhost;
    
    location / {
        root   /usr/share/nginx/html;
        index  index.html index.htm;
    }
}
EOF

# 设置 hdfs 实例的名称
$ export HDFS_POD=$(kubectl get pod -n kuda-system -l app=hdfs -o jsonpath={.items..metadata.name})

# 将本地配置文件拷贝到容器中
$ kubectl cp test.conf -n kuda-system $HDFS_POD:/opt/hadoop

# 在 hdfs 服务端创建目录
$ kubectl exec -n kuda-system $HDFS_POD -- hadoop fs -mkdir -p /kuda/conf

# 将配置数据上传到 hdfs 服务端
$ kubectl exec -n kuda-system $HDFS_POD -- hadoop fs -put /opt/hadoop/test.conf /kuda/conf
```

3、检查配置文件是否上传成功
```shell
$ kubectl exec -n kuda-system $HDFS_POD -- hadoop fs -ls /kuda/conf
Found 1 items
-rw-r--r--   1 root supergroup        165 2021-10-29 09:51 /kuda/conf/test.conf
```

## 操作步骤

1、 提交 DataSet 内容
```yaml
$ cat <<EOF | kubectl apply -f -
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
EOF
```

2、创建 nginx 示例应用
```yaml
$ cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
            - containerPort: 80
EOF
```

3、检查服务是否成功启动

```shell
$ kubectl get pod
NAME                                READY   STATUS    RESTARTS   AGE
nginx-deployment-66b6c48dd5-n69ps   2/2     Running   0          15s
```
该实例已经自动注入 sidecar 容器，并成功启动。

4、检查数据下发状态

```shell
$ kubectl get dataset
NAME            DATAITEMS   READY   AGE
dataset-nginx   1           1/1     41s

$ kubectl get data
NAME                             READY   AGE
dataset-nginx-66b6c48dd5-n69ps   1/1     35s
```

可以看到，模型数据已经成功下发。

5、验证服务是否符合预期

执行下面的命令验证 nginx 服务的 8081 端口是否正常工作:
```shell
$ export NGINX_POD_IP=$(kubectl get pod -l app=nginx -o jsonpath={.items..status.podIP})
$ kubectl run debug -i --rm --quiet=true --restart=Never --image=curlimages/curl -- curl -o /dev/null -s -w "%{http_code}\n" http://$NGINX_POD_IP:8081
200
```

更进一步，可以修改 test.conf 配置文件内容，并将修改后的文件上传到 HDFS 存储端，同时更新 dataset 资源，以验证文件更新和生效的过程。

## 清理

```shell
$ kubectl delete dataset dataset-nginx
$ kubectl delete deployment nginx-deployment
```