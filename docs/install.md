# 安装说明

目前支持以下两种方式进行安装:

* 方式一: 通过 kubectl 快速安装
* 方式二: 通过源码编译安装


## 安装步骤

### 准备

* Kubernetes 集群，版本不低于1.16，支持 CRD 和 AdmissionWebhook
* Kubectl 工具
* Git 工具 (使用方式二)

### 通过 kubectl 快速安装

在准备好的 Kubernetes 集群上，执行以下命令即可快速完成安装:

```bash
kubectl apply -f https://raw.githubusercontent.com/kuda-io/kuda/master/install/kuda.yaml
```

### 通过源码编译安装

1、下载项目代码

```shell
git clone https://github.com/kuda-io/kuda.git
cd kuda
```

2、项目编译

支持使用 Makefile 进行项目编译，部分编译命令如下:
```shell
# 编译所有组件
make build

# 只编译 manager 组件
make build-manager

# 只编译 webhook 组件
make build-webhook

# 编译所有组件的镜像，其中 MANAGER_IMG 和 WEBHOOK_IMG 分别设置为 manager 和 webhook 组件的镜像名称
make docker-build MANAGER_IMG={xxx} WEBHOOK_IMG={xxx}

# 将编译好的镜像推送到镜像仓库
make docker-push MANAGER_IMG={xxx} WEBHOOK_IMG={xxx}
```
                                                 
3、 安装

编译完成后，执行以下命令即可安装:
```shell
make deploy MANAGER_IMG={xxx} WEBHOOK_IMG={xxx}    
```

## 验证

安装完成后，验证各个组件的状态是否符合预期:
```shell
$ kubectl get all -n kuda-system
NAME                                           READY   STATUS      RESTARTS   AGE
pod/kuda-controller-manager-754b654b75-5q25s   2/2     Running     0          52s
pod/kuda-webhook-59df7dc545-275cn              1/1     Running     0          51s
pod/kuda-webhook-init-7hkwl                    0/1     Completed   0          51s

NAME                                              TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
service/kuda-controller-manager-metrics-service   ClusterIP   10.96.27.64     <none>        8443/TCP   53s
service/kuda-webhook                              ClusterIP   10.96.180.222   <none>        443/TCP    53s

NAME                                      READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/kuda-controller-manager   1/1     1            1           53s
deployment.apps/kuda-webhook              1/1     1            1           52s

NAME                                                 DESIRED   CURRENT   READY   AGE
replicaset.apps/kuda-controller-manager-754b654b75   1         1         1       53s
replicaset.apps/kuda-webhook-59df7dc545              1         1         1       52s

NAME                          COMPLETIONS   DURATION   AGE
job.batch/kuda-webhook-init   1/1           6s         52s
```

以上状态说明各组件已经正常运行，至此，kuda 安装成功。

> 说明: 在使用过程中，数据将统一放到 `/kuda/data` 目录，举例来说，如果您在 DataSet 中设置的 localPath 为 `/models/half_plus_two`，则数据下载的最终目录是 `/kuda/data/models/half_plus_two`。
该基础目录支持自定义配置，您可以通过命令`kubectl edit configmaps -n kuda-system kuda-webhook-config`进行编辑，修改配置中的 dataPathPrefix 字段即可。


## 安装附加组件 (可选)

为了方便您快速体验 Kuda 产品功能，我们准备了 HDFS 存储组件，您可以通过如下命令选择安装:
```shell
kubectl apply -f https://raw.githubusercontent.com/kuda-io/kuda/master/install/addons/hdfs.yaml
```
检查组件是否正常运行:
```shell
$ kubectl get pod -n kuda-system -l app=hdfs
NAME                              READY   STATUS    RESTARTS   AGE
hdfs-deployment-889d87db7-qlds2   1/1     Running   0          27s
```

> 注意：该组件只供测试使用，无法用于生产环境。

## 卸载

执行如下命令卸载 kuda 组件:

```shell
# 卸载附件组件
kubectl delete -f https://raw.githubusercontent.com/kuda-io/kuda/master/install/addons/hdfs.yaml

# 卸载平台组件
kubectl delete -f https://raw.githubusercontent.com/kuda-io/kuda/master/install/kuda.yaml
```