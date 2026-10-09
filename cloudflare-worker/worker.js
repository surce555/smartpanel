/**
 * SmartPanel Cloudflare Worker - 智能双栈内外网分流
 * 
 * 逻辑说明：
 * 1. 客户端访问主域名 (如 pan.yourdomain.com) 时，Worker 首先读取 CF-Connecting-IP。
 * 2. 如果包含 ":" (说明客户端具备 IPv6 环境):
 *    - 检查是否带有 redirected=1 参数（避免重复重定向产生死循环）。
 *    - (可选) 短超时探活 NAS IPv6 直连端口；若通畅则返回 302 重定向到直连域名。
 *    - 若不通或超时，无缝回退走 Cloudflare Tunnel。
 * 3. 如果是 IPv4 客户端，直接 fetch(request) 走 Tunnel 回源。
 */

export default {
  async fetch(request, env) {
    const url = new URL(request.url);

    // 1. 如果已携带 redirected=1 参数，直接放行，避免重定向循环
    if (url.searchParams.has('redirected')) {
      return fetch(request);
    }

    // 2. 获取客户端真实 IP
    const clientIP = request.headers.get('cf-connecting-ip') || '';
    const isIPv6 = clientIP.includes(':');

    // 3. 配置的目标 IPv6 直连域名及端口 (可由环境变量 V6_TARGET 覆盖，如 "v6.yourdomain.com:5666")
    const v6Target = env.V6_TARGET || 'v6.yourdomain.com:5666';

    if (isIPv6 && v6Target) {
      // 构造直连目标 URL，追加 redirected=1 标志
      const targetUrl = new URL(url.pathname + url.search, `https://${v6Target}`);
      targetUrl.searchParams.set('redirected', '1');

      // 可选：探活 NAS IPv6 端点，超时设为 400ms
      const probeEnabled = env.ENABLE_PROBE !== 'false';
      if (probeEnabled) {
        try {
          const probeController = new AbortController();
          const timeoutId = setTimeout(() => probeController.abort(), 400);

          const probeUrl = `https://${v6Target}/api/health/ping`;
          const probeResp = await fetch(probeUrl, {
            method: 'GET',
            signal: probeController.signal,
          });
          clearTimeout(timeoutId);

          if (probeResp.ok) {
            return Response.redirect(targetUrl.toString(), 302);
          }
        } catch (e) {
          // 探测失败或超时，自动回退走 Tunnel，不中断用户访问
          return fetch(request);
        }
      } else {
        return Response.redirect(targetUrl.toString(), 302);
      }
    }

    // IPv4 用户或回退请求：直接通过 Cloudflare Tunnel 回源
    return fetch(request);
  },
};
