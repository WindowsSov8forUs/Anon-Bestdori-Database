package config

const DEFAULT_CONFIG = `
log:
  level: info

mongo:
  uri: "mongodb://localhost:27017/"

api:
  timeout: 5
  proxy: ""

server:
  host: "0.0.0.0"
  port: 8080
`
