declare namespace API {
  type basicHeartbeatParams = {
    /** 服务名称 */
    service_name: string;
    /** 通讯密钥 */
    secret: string;
  };

  type HeartbeatRequest = true;

  type HeartbeatResponse = {
    /** 心跳是否成功 */
    status: boolean;
    /** 返回信息 */
    message?: string;
    /** 服务端时间戳 */
    timestamp?: number;
  };

  type RegisterServiceRequest = {
    /** 心跳地址 */
    heartbeat_url: string;
    /** 代理路径 */
    proxy_path: string;
    /** 服务代码仓库 */
    repository: string;
    /** 通讯密钥 */
    secret: string;
    /** 服务名称 */
    service_name: string;
    /** 服务地址 */
    service_url: string;
    /** 服务版本 */
    service_version: string;
  };

  type RegisterServiceResponse = {
    /** 返回信息 */
    message: string;
    /** 注册是否成功 */
    success: boolean;
  };
}
