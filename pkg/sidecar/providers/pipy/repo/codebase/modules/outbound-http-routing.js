((
  config = pipy.solve('config.js'),

  allMethods = ['GET', 'HEAD', 'POST', 'PUT', 'DELETE', 'PATCH'],

  funcFailover = json => (
    json ? ((obj = null) => (
      obj = Object.fromEntries(
        Object.entries(json).map(
          ([k, v]) => (
            (v === 0) ? ([k, 1]) : null
          )
        ).filter(e => e)
      ),
      Object.keys(obj).length === 0 ? null : new algo.RoundRobinLoadBalancer(obj)
    ))() : null
  ),

  clusterCache = new algo.Cache(
    (clusterName => (
      (cluster = config?.Outbound?.ClustersConfigs?.[clusterName]) => (
        cluster ? Object.assign({ name: clusterName }, cluster) : null
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
            matchPath = (
              (config.Type === 'Regex') && (
                ((match = null) => (
                  match = new RegExp(config.Path),
                  (path) => match.test(path)
                ))()
              ) || (config.Type === 'Exact') && (
                (path) => path === config.Path
              ) || (config.Type === 'Prefix') && (
                (path) => path.startsWith(config.Path)
              )
            ),
            headerRules = config.Headers ? Object.entries(config.Headers).map(([k, v]) => [k, new RegExp(v)]) : null,
            balancer = new algo.RoundRobinLoadBalancer(config.TargetClusters || {}),
            failoverBalancer = funcFailover(config.TargetClusters),
            service = Object.assign({ name: serviceName }, portConfig.HttpServiceRouteRules[serviceName]),
            rule = headerRules ? (
              (path, headers) => matchPath(path) && headerRules.every(([k, v]) => v.test(headers[k] || '')) && (
                __route = config,
                __service = service,
                __cluster = clusterCache.get(balancer.next()?.id),
                failoverBalancer && (
                  _failoverCluster = clusterCache.get(failoverBalancer.next()?.id)
                )
              )
            ) : (
              (path) => matchPath(path) && (
                __route = config,
                __service = service,
                __cluster = clusterCache.get(balancer.next()?.id),
                failoverBalancer && (
                  _failoverCluster = clusterCache.get(failoverBalancer.next()?.id)
                )
              )
            ),
            allowedMethods = config.Methods || allMethods,
          ) => (
            allowedMethods.forEach(
              method => (tree[method] || (tree[method] = [])).push(rule)
            )
          )
        )()
      ),

      (method, path, headers) => void (
        tree[method]?.find?.(rule => rule(path, headers)),
        __cluster && (
          headers['serviceidentity'] = __port.ServiceIdentity
        )
      )
    )
  )(),

  makePortHandler = (portConfig) => (
    (
      serviceHandlers = new algo.Cache(
        (serviceName) => makeServiceHandler(portConfig, serviceName)
      ),

      hostHandlers = new algo.Cache(
        (host) => serviceHandlers.get(portConfig.HttpHostPort2Service[host])
      ),
    ) => (
      (msg) => (
        (
          head = msg.head,
          headers = head.headers,
        ) => (
          hostHandlers.get(headers.host)(head.method, head.path, headers)
        )
      )()
    )
  )(),

  portHandlers = new algo.Cache(makePortHandler),

) => pipy({
  _failoverCluster: null,
})

.import({
  __port: 'outbound-main',
  __cluster: 'outbound-main',
})

.export('outbound-http-routing', {
  __route: null,
  __service: null,
})

.pipeline()
.demuxHTTP().to(
  $=>$
  .replay({ 'delay': 0 }).to(
    $=>$
    .handleMessageStart(
      msg => (
        _failoverCluster && (
          __cluster = _failoverCluster,
          _failoverCluster = null,
          true
        ) || (
          portHandlers.get(__port)(msg)
        )
      )
    )
    .chain()
    .replaceMessage(
      msg => (
        (
          status = msg?.head?.status
        ) => (
          _failoverCluster && (!status || status < '200' || status > '399') ? new StreamEnd('Replay') : msg
        )
      )()
    )
  )
)

)()
