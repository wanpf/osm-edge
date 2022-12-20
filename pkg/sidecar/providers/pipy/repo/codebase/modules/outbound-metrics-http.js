((
  {
    clusterCache,
    identityCache,
  } = pipy.solve('modules/metrics.js'),
) => (

pipy({
  _requestTime: null
})

.import({
  __cluster: 'outbound-main'
})

.pipeline()
.handleMessageStart(
  () => (
    _requestTime = Date.now()
  )
)
.chain()
.handleMessageStart(
  (msg) => (
    (
      clusterName = __cluster?.name,
      status = msg?.head?.status,
      statusClass = status / 100,
      metrics = clusterCache.get(clusterName),
      osmRequestDurationHist = identityCache.get(msg?.head?.headers?.['osm-stats']),
    ) => (
      osmRequestDurationHist?.observe?.(Date.now() - _requestTime),
      metrics.upstreamCompletedCount.increase(),
      metrics.upstreamResponseTotal.increase(),
      status && (
        metrics.upstreamCodeCount.withLabels(status).increase(),
        metrics.upstreamCodeXCount.withLabels(statusClass).increase(),
        metrics.upstreamResponseCode.withLabels(statusClass).increase()
      )
    )
  )()
)

))()