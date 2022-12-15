((
  config = pipy.solve('config.js'),

  certChain = config?.Certificate?.CertChain,
  privateKey = config?.Certificate?.PrivateKey,
  issuingCA = config?.Certificate?.IssuingCA,

  listIssuingCA = (
    (cas = []) => (
      issuingCA && cas.push(new crypto.Certificate(issuingCA)),
      Object.values(config?.Outbound?.TrafficMatches || {}).map(
        a => a.map(
          o => Object.values(o.DestinationIPRanges || {}).map(
            c => c?.SourceCert?.IssuingCA && (
              cas.push(new crypto.Certificate(c?.SourceCert?.IssuingCA))
            )
          )
        )
      ),
      cas
    )
  )(),

  forwardMatches = config?.Forward?.ForwardMatches && Object.fromEntries(
    Object.entries(
      forwardMatches).map(
        ([k, v]) => [
          k, new algo.RoundRobinLoadBalancer(v || {})
        ]
      )
  ),

  forwardEgressGateways = config?.Forward?.EgressGateways && Object.fromEntries(
    Object.entries(
      forwardEgressGateways).map(
        ([k, v]) => [
          k, new algo.RoundRobinLoadBalancer(v?.Endpoints || {})
        ]
      )
  ),
) => (

pipy({
  _egressEndpoint: null,
})

.import({
  __port: 'outbound-main',
  __cert: 'outbound-main',
  __address: 'outbound-main',
  __egressEnable: 'outbound-main',
})

.pipeline()
.onStart(
  () => void (
    forwardMatches && ((egw = forwardMatches[__port?.EgressForwardGateway || '*']?.next?.()?.id) => (
      egw && (_egressEndpoint = forwardEgressGateways?.[egw]?.next?.()?.id)
    ))(),
    console.log('outbound - TLS/__egressEnable/_egressEndpoint/__cert/__address:', Boolean(certChain), __egressEnable, _egressEndpoint, Boolean(__cert), __address)
  )
)
.branch(
  () => !__address, $=>$.chain(),

  () => __cert, (
    $=>$
    .connectTLS({
      certificate: () => ({
        cert: new crypto.Certificate(__cert.CertChain),
        key: new crypto.PrivateKey(__cert.PrivateKey),
      }),
      trusted: listIssuingCA,
    }).to($=>$.connect(() => __address))
  ),

  () => certChain && !__egressEnable, (
    $=>$
    .connectTLS({
      certificate: {
        cert: new crypto.Certificate(certChain),
        key: new crypto.PrivateKey(privateKey),
      },
      trusted: issuingCA ? [new crypto.Certificate(issuingCA)] : [],
    }).to($=>$.connect(() => __address))
  ),

  () => __egressEnable && _egressEndpoint, (
    $=>$
    .connectSOCKS(
      () => __address,
    ).to($=>$.connect(() => _egressEndpoint))
  ),

  $=>$.connect(() => __address)
)

))()