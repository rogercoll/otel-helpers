package nginxotel

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	"gopkg.in/yaml.v3"
)

type ProcessorInstance struct {
	processorType component.ID
	config        string
}

type ProcessorsPipeline struct {
	signal  pipeline.Signal
	configs []ProcessorInstance
	yaml.Marshaler
}

var IngressNginxProcessors = []ProcessorInstance{
	{
		processorType: component.MustNewIDWithName("transform", "parse_nginx_ingress_access/log"),
		config: `
    error_mode: ignore
    log_statements:
      - context: log
        conditions:
            #     # ^([0-9a-fA-F:.]+): Matches the remote address (IPv4 or IPv6 format).
            #     # [^ ]+: Matches the remote user (including the hyphen for missing user).
            #     # .*[0-9a-fA-F]+$: Ensures the log line ends with a hexadecimal string (request ID).
          - IsMatch(body, "^([0-9a-fA-F:.]+) - [^ ]+ .*[0-9a-fA-F]+$")
        statements:
          # Log format: https://github.com/kubernetes/ingress-nginx/blob/nginx-0.30.0/docs/user-guide/nginx-configuration/log-format.md
          # Based on https://github.com/elastic/integrations/blob/main/packages/nginx_ingress_controller/data_stream/access/elasticsearch/ingest_pipeline/default.yml
          - set(body, ExtractGrokPatterns(body, "(%{NGINX_HOST} )?\"?(?:%{NGINX_ADDRESS_LIST:nginx_ingress_controller.access.remote_ip_list}|%{NOTSPACE:source.address}) - (-|%{DATA:user.name}) \\[%{HTTPDATE:nginx_ingress_controller.access.time}\\] \"%{DATA:nginx_ingress_controller.access.info}\" %{NUMBER:http.response.status_code:long} %{NUMBER:http.response.body.size:long} \"(-|%{DATA:http.request.referrer})\" \"(-|%{DATA:user_agent.original})\" %{NUMBER:http.request.size:long} %{NUMBER:http.request.time:double} \\[%{DATA:upstream.name}\\] \\[%{DATA:upstream.alternative_name}\\] (%{UPSTREAM_ADDRESS_LIST:upstream.address}|-) (%{UPSTREAM_RESPONSE_SIZE_LIST:upstream.response.size_list}|-) (%{UPSTREAM_RESPONSE_TIME_LIST:upstream.response.time_list}|-) (%{UPSTREAM_RESPONSE_STATUS_CODE_LIST:upstream.response.status_code_list}|-) %{GREEDYDATA:http.request.id}", true, ["NGINX_HOST=(?:%{IP:destination.ip}|%{NGINX_NOTSEPARATOR:destination.domain})(:%{NUMBER:destination.port})?", "NGINX_NOTSEPARATOR=[^\t ,:]+", "NGINX_ADDRESS_LIST=(?:%{IP}|%{WORD}) (\"?,?\\s*(?:%{IP}|%{WORD}))*", "UPSTREAM_ADDRESS_LIST=(?:%{IP}(:%{NUMBER})?)(\"?,?\\s*(?:%{IP}(:%{NUMBER})?))*", "UPSTREAM_RESPONSE_SIZE_LIST=(?:%{NUMBER})(\"?,?\\s*(?:%{NUMBER}))*", "UPSTREAM_RESPONSE_TIME_LIST=(?:%{NUMBER})(\"?,?\\s*(?:%{NUMBER}))*", "UPSTREAM_RESPONSE_STATUS_CODE_LIST=(?:%{NUMBER})(\"?,?\\s*(?:%{NUMBER}))*", "IP=(?:\\[?%{IPV6}\\]?|%{IPV4})"]))
          - merge_maps(body, ExtractGrokPatterns(body["nginx_ingress_controller.access.info"], "%{WORD:http.request.method} %{DATA:url.original} HTTP/%{NUMBER:http.version}", true), "upsert")
          - delete_key(body, "nginx_ingress_controller.access.info")

          # Extra URL parsing
          - merge_maps(body, URL(body["url.original"]), "upsert")
          - set(body["url.domain"], body["destination.domain"])

          # set source.address as attribute for GeoIP processor
          - set(attributes["source.address"], body["source.address"])

          - set(attributes["data_stream.dataset"], "nginx_ingress_controller.access")

          # LogRecord event: https://github.com/open-telemetry/semantic-conventions/pull/982
          - set(attributes["event.name"], "nginx_ingress_controller.access")
          - set(attributes["event.timestamp"], String(Time(body["nginx_ingress_controller.access.time"], "%d/%b/%Y:%H:%M:%S %z")))

          - delete_key(body, "nginx_ingress_controller.access.time")

      - context: log
        conditions:
            # Extract user agent when not empty
          - body["user_agent.original"] != nil
        statements:
          # Extract UserAgent
          # TODO: UserAgent OTTL function does not provide os specific metadata yet: https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/35458
          - merge_maps(body, UserAgent(body["user_agent.original"]), "upsert")

      - context: log
        conditions:
          - body["upstream.response.time_list"] != nil
        statements:
          # Extract comma separated list
          # TODO: We would like to get the sum over all upstream.response.time_list values instead of providing a slice with all the values
          - set(body["upstream.response.time"], Split(body["upstream.response.time_list"], ","))
          - delete_key(body, "upstream.response.time_list")

      - context: log
        conditions:
          - body["upstream.response.size_list"] != nil
        statements:
          # Extract comma separated list
          # TODO: We would like to get the Last upstream.response.size_list value instead of providing a slice with all the values
          # See: https://github.com/elastic/integrations/blob/main/packages/nginx_ingress_controller/data_stream/access/elasticsearch/ingest_pipeline/default.yml#L94b
          - set(body["upstream.response.size"], Split(body["upstream.response.size_list"], ","))
          - delete_key(body, "upstream.response.size_list")

      - context: log
        conditions:
          - body["upstream.response.status_code_list"] != nil
        statements:
          # Extract comma separated list
          # TODO: We would like to get the Last upstream.response.status_code_list value instead of providing a slice with all the values
          - set(body["upstream.response.status_code"], Split(body["upstream.response.status_code_list"], ","))
          - delete_key(body, "upstream.response.status_code_list")
`,
	},
	{
		processorType: component.MustNewIDWithName("transform", "parse_nginx_ingress_error/log"),
		config: `
    error_mode: ignore
    log_statements:
      - context: log
        conditions:
            # ^[EWF]: Matches logs starting with E (Error), W (Warning), or F (Fatal).
            # \d{4}: Matches the four digits after the log level (representing the date, like 1215 for December 15).
            # .+: Matches the rest of the log line (the message part, without needing specific timestamp or file format).
          - IsMatch(body, "^[EWF]\\d{4} .+")
        statements:
          - set(body, ExtractGrokPatterns(body, "%{LOG_LEVEL:log.level}%{MONTHNUM}%{MONTHDAY} %{HOUR}:%{MINUTE}:%{SECOND}\\.%{MICROS}%{SPACE}%{NUMBER:thread_id} %{SOURCE_FILE:source.file.name}:%{NUMBER:source.line_number}\\] %{GREEDYMULTILINE:message}", true, ["LOG_LEVEL=[A-Z]", "MONTHNUM=(0[1-9]|1[0-2])", "MONTHDAY=(0[1-9]|[12][0-9]|3[01])", "HOUR=([01][0-9]|2[0-3])", "MINUTE=[0-5][0-9]", "SECOND=[0-5][0-9]", "MICROS=[0-9]{6}", "SOURCE_FILE=[^:]+", "GREEDYMULTILINE=(.|\\n)*"]))

          - set(attributes["data_stream.dataset"], "nginx_ingress_controller.error")

          # LogRecord event: https://github.com/open-telemetry/semantic-conventions/pull/982
          - set(attributes["event.name"], "nginx_ingress_controller.error")
`,
	},
}

var IngressNginxPipeline = ProcessorsPipeline{
	signal:  pipeline.SignalLogs,
	configs: IngressNginxProcessors,
}

// TODO: return collector's ready configuration
// yaml.MarshalYAML interface
func (p *ProcessorsPipeline) MarshalYAML() (interface{}, error) {
	return nil, errors.New("TODO")
}
