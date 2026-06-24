import http2 from 'node:http2'
import { defineConfig, loadEnv } from 'vite'
import uniPlugin from '@dcloudio/vite-plugin-uni'

const uni = uniPlugin.default || uniPlugin

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const apiProxyTarget = env.VITE_API_BASE 
  const debugCustomerToken = env.VITE_CUSTOMER_TOKEN || ''
  return {
    plugins: [h2ApiProxy(apiProxyTarget, debugCustomerToken), uni()],
    server: {
      proxy: {
        '/api': {
          target: apiProxyTarget,
          changeOrigin: true,
          configure: (proxy) => {
            proxy.on('proxyReq', (proxyReq) => {
              proxyReq.removeHeader('origin')
              if (debugCustomerToken) proxyReq.setHeader('token', debugCustomerToken)
            })
          }
        }
      }
    }
  }
})

function h2ApiProxy(target, debugCustomerToken) {
  const endpoint = new URL(target)
  if (endpoint.protocol !== 'https:') return { name: 'h2-api-proxy' }
  return {
    name: 'h2-api-proxy',
    configureServer(server) {
      server.middlewares.use('/api', async (req, res) => {
        try {
          const chunks = []
          for await (const chunk of req) chunks.push(chunk)
          const body = chunks.length ? Buffer.concat(chunks) : null
          const client = http2.connect(endpoint.origin)
          const headers = {
            ':method': req.method || 'GET',
            ':path': req.originalUrl || req.url || '/api',
            ':scheme': endpoint.protocol.slice(0, -1),
            ':authority': endpoint.host,
            accept: req.headers.accept || 'application/json'
          }
          const contentType = req.headers['content-type']
          if (contentType) headers['content-type'] = contentType
          const token = debugCustomerToken || req.headers.token
          if (token) headers.token = token
          if (body) headers['content-length'] = String(body.length)
          const proxyReq = client.request(headers)
          proxyReq.on('response', (proxyHeaders) => {
            const status = Number(proxyHeaders[':status'] || 502)
            res.statusCode = status
            Object.entries(proxyHeaders).forEach(([key, value]) => {
              if (key.startsWith(':') || value === undefined) return
              res.setHeader(key, value)
            })
          })
          proxyReq.on('data', (chunk) => res.write(chunk))
          proxyReq.on('end', () => {
            res.end()
            client.close()
          })
          proxyReq.on('error', (error) => {
            if (!res.headersSent) res.statusCode = 502
            res.end(JSON.stringify({ message: error.message || 'H5 API proxy failed' }))
            client.close()
          })
          if (body) proxyReq.end(body)
          else proxyReq.end()
        } catch (error) {
          res.statusCode = 502
          res.end(JSON.stringify({ message: error.message || 'H5 API proxy failed' }))
        }
      })
    }
  }
}
