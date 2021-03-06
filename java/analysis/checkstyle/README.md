## 组件名称：Java Checkstyle Analysis

使用 Checkstyle 对 java 项目进行代码检查

### 组件参数
#### 入参

* `GIT_CLONE_URL` 必填，源代码地址，如为私有仓库需要授权; 如需使用系统关联的git仓库, 可以从系统提供的全局环境变量中获取: ${_WORKFLOW_GIT_CLONE_URL}
* `GIT_REF` 非必填，源代码git目标引用，可以是一个git branch, git tag 或者git commit ID, 默认值master
* `ANALYSIS_TARGET` 非必填，目标文件路径, 默认为项目下所有java文件
* `ANALYSIS_OPTIONS` 非必填，如 `--debug`

#### 出参

无

### Tag列表及其Dockerfile链接

* 8.10, latest: [Dockerfile]()

### 源码地址

Java Checkstyle Analysis：<https://github.com/tencentyun/workflow-components/tree/master/java/analysis/checkstyle>

### 构建

`docker build -t hub.tencentyun.com/tencenthub/java_analysis_checkstyle:latest .`
