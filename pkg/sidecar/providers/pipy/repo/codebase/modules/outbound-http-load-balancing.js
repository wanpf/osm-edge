((
  retryCounter = new stats.Counter('sidecar_cluster_upstream_rq_retry', ['sidecar_cluster_name']),
  retrySuccessCounter = new stats.Counter('sidecar_cluster_upstream_rq_retry_success', ['sidecar_cluster_name']),
  retryLimitCounter = new stats.Counter('sidecar_cluster_upstream_rq_retry_limit_exceeded', ['sidecar_cluster_name']),

  makeClusterConfig = (clusterConfig) => (
    clusterConfig &&
    {
      targetBalancer: new algo.RoundRobinLoadBalancer(
        Object.fromEntries(Object.entries(clusterConfig.Endpoints).map(([k, v]) => [k, v.Weight || 100]))
      ),
      needRetry: Boolean(clusterConfig.RetryPolicy?.NumRetries),
      numRetries: clusterConfig.RetryPolicy?.NumRetries,
      retryStatusCodes: (clusterConfig.RetryPolicy?.RetryOn || '5xx').split(',').reduce(
        (lut, code) => (
          code.endsWith('xx') ? (
            new Array(100).fill(0).forEach((_, i) => lut[(code.charAt(0)|0)*100+i] = true)
          ) : (
            lut[code|0] = true
          ),
          lut
        ),
        []
      ),
      retryBackoffBaseInterval: clusterConfig.RetryPolicy?.RetryBackoffBaseInterval || 1, // default 1 second
      retryCounter: retryCounter.withLabels(clusterConfig.name),
      retrySuccessCounter: retrySuccessCounter.withLabels(clusterConfig.name),
      retryLimitCounter: retryLimitCounter.withLabels(clusterConfig.name),
      muxHttpOptions: {
        version: () => __isHTTP2 ? 2 : 1,
        maxMessages: clusterConfig.ConnectionSettings?.http?.MaxRequestsPerConnection
      },
    }
  ),

  clusterConfigs = new algo.Cache(makeClusterConfig),

  shouldRetry = (statusCode) => (
    _clusterConfig.retryStatusCodes[statusCode] ? (
      (_retryCount < _clusterConfig.numRetries) ? (
        _clusterConfig.retryCounter.increase(),
        _retryCount++,
        true
      ) : (
        _clusterConfig.retryLimitCounter.increase(),
        false
      )
    ) : (
      _retryCount > 0 && _clusterConfig.retrySuccessCounter.increase(),
      false
    )
),

) => pipy({
  _retryCount: 0,
  _clusterConfig: null,
})

.import({
  __isHTTP2: 'outbound-main',
  __cluster: 'outbound-main',
  __target: 'outbound-main',
})

.export('outbound-http-load-balancing', {
  __targetObject: null,
  __muxHttpOptions: null,
})

.pipeline()
.onStart(
  () => void (
    (_clusterConfig = clusterConfigs.get(__cluster)) && (
      __targetObject = _clusterConfig.targetBalancer?.next?.(),
      __target = __targetObject?.id,
      __muxHttpOptions = _clusterConfig.muxHttpOptions
    )
  )
)
.branch(
  () => _clusterConfig.needRetry, (
    $=>$
    .replay({
        delay: () => _clusterConfig.retryBackoffBaseInterval * Math.min(10, Math.pow(2, _retryCount-1)|0)
    }).to(
      $=>$
      .chain()
      .replaceMessageStart(
        msg => (
          shouldRetry(msg.head.status) ? new StreamEnd('Replay') : msg
        )
      )
    )
  ),

  (
    $=>$.chain()
  )
)

)()