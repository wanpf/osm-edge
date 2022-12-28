((
  {
    identity,
    clusterCache,
  } = pipy.solve('metrics.js'),
) => (

pipy()

.import({
  __cluster: 'inbound',
})

.pipeline()
.chain()
.handleMessageStart(
  (msg) => (
    (
      headers = msg?.head?.headers,
      metrics = clusterCache.get(__cluster?.name),
    ) => (
      headers && (
        headers['osm-stats'] = identity,
        metrics.upstreamResponseTotal.increase(),
        metrics.upstreamResponseCode.withLabels(msg?.head?.status / 100).increase()
      )
    )
  )()
)

))()