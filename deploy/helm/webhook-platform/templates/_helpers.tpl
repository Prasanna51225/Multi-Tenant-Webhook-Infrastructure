{{- define "webhook-platform.name" -}}
{{- .Chart.Name -}}
{{- end -}}

{{- define "webhook-platform.fullname" -}}
{{- .Release.Name -}}
{{- end -}}

{{- define "webhook-platform.labels" -}}
app.kubernetes.io/name: {{ include "webhook-platform.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}