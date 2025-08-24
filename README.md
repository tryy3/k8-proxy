# K8-Proxy
## TODO
 * Make it either tsnet only or docker tailscale only
    * With the current setup I have 2 different connections that require tailscale, the input (tsnet currently) and kubernetes API requests (regular connection).
    In best scenario situation I'd like to make use of tsnet only, this way I can simply contain tailscale inside the app and not the container.
    It might be possible using HTTP requests with tsnet, skip or inject my own HTTP client into the kubernetes API