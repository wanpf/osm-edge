((
  config = pipy.solve('config.js'),

  allMethods = ['GET', 'HEAD', 'POST', 'PUT', 'DELETE', 'PATCH'],

  clusterCache = new algo.Cache(
    (clusterName => (
      (cluster = config?.Inbound?.ClustersConfigs?.[clusterName]) => (
        cluster ? Object.assign({ clusterName, Endpoints: cluster }) : null
      )
    )())
  ),

  makeServiceHandler = (portConfig, serviceName) => (
    (
      rules = portConfig.HttpServiceRouteRules[serviceName]?.RouteRules || [],
      tree = {},
    ) => (
      rules.forEach(
        config => (
          (
            matchPath = ((match = null) => (
              (config.Type === 'Regex') && (
                match = new RegExp(config.Path),
                (path) => match.test(path)
              ) || (config.Type === 'Exact') && (
                (path) => path === config.Path
              ) || (config.Type === 'Prefix') && (
                (path) => path.startsWith(config.Path)
              )
            ))(),
            headerRules = config.Headers ? Object.entries(config.Headers).map(([k, v]) => [k, new RegExp(v)]) : null,
            balancer = new algo.RoundRobinLoadBalancer(config.TargetClusters || {}),
            service = Object.assign({ serviceName }, portConfig.HttpServiceRouteRules[serviceName]),
            rule = headerRules ? (
              (path, headers) => matchPath(path) && headerRules.every(([k, v]) => v.test(headers[k] || '')) && (
                __route = config,
                __service = service,
                ((clusterName = balancer.next()?.id) => (
                  __cluster = clusterCache.get(clusterName)
                ))()
              )
            ) : (
              (path) => matchPath(path) && (
                __route = config,
                __service = service,
                ((clusterName = balancer.next()?.id) => (
                  __cluster = clusterCache.get(clusterName)
                ))()
              )
            ),
            allowedIdentities = config.AllowedServices ? new Set(config.AllowedServices) : [''],
            allowedMethods = config.Methods || allMethods,
          ) => (
            allowedIdentities.forEach(
              identity => (
                (
                  methods = tree[identity] || (tree[identity] = {}),
                ) => (
                  allowedMethods.forEach(
                    method => (methods[method] || (methods[method] = [])).push(rule)
                  )
                )
              )()
            )
          )
        )()
      ),

      (method, path, headers) => void (
        tree[headers.serviceidentity || '']?.[method]?.find?.(rule => rule(path, headers))
      )
    )
  )(),

  makePortHandler = portConfig => (
    (
      ingressRanges = Object.keys(portConfig.SourceIPRanges || {}).map(k => new Netmask(k)),

      serviceHandlers = new algo.Cache(
        serviceName => makeServiceHandler(portConfig, serviceName)
      ),

      makeHostHandler = (portConfig, host) => (
        serviceHandlers.get(portConfig.HttpHostPort2Service[host])
      ),

      hostHandlers = new algo.Cache(
        host => makeHostHandler(portConfig, host)
      ),
    ) => (
      ingressRanges.length > 0 ? (
        msg => void(
          (
            ip = __inbound.remoteAddress || '127.0.0.1',
            ingressRange = ingressRanges.find(r => r.contains(ip)),
            head = msg.head,
            headers = head.headers,
            handler = hostHandlers.get(ingressRange ? '*' : headers.host),
          ) => (
            __ingressEnable = Boolean(ingressRange),
            handler(head.method, head.path, headers)
          )
        )()
      ) : (
        msg => void (
          (
            head = msg.head,
            headers = head.headers,
            handler = hostHandlers.get(headers.host),
          ) => (
            handler(head.method, head.path, headers)
          )
        )()
      )
    )
  )(),

  portHandlers = new algo.Cache(makePortHandler),

) => pipy()

.import({
  __port: 'inbound-main',
  __cluster: 'inbound-main',
})

.export('inbound-http-routing', {
  __route: null,
  __service: null,
  __ingressEnable: false,
})

.pipeline()
.demuxHTTP().to(
  $=>$
  .handleMessageStart(
    msg => portHandlers.get(__port)(msg)
  )
  .chain()
)

)()