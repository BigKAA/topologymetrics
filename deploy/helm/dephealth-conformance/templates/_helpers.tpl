{{/*
Путь к custom-образу: pushRegistry/image:tag или image:tag
*/}}
{{- define "dephealth-conformance.image" -}}
{{- if .registry -}}
{{ .registry }}/{{ .image }}:{{ .tag }}
{{- else -}}
{{ .image }}:{{ .tag }}
{{- end -}}
{{- end -}}

{{/*
Общие метки.
*/}}
{{- define "dephealth-conformance.labels" -}}
app.kubernetes.io/part-of: dephealth
app.kubernetes.io/component: conformance
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
{{- end -}}

{{/*
Метки селектора для конкретного сервиса.
*/}}
{{- define "dephealth-conformance.selectorLabels" -}}
app: {{ .name }}
app.kubernetes.io/name: {{ .name }}
app.kubernetes.io/part-of: dephealth
{{- end -}}

{{/*
Целевой namespace (переопределяет dephealth-infra namespace).
*/}}
{{- define "dephealth-conformance.namespace" -}}
{{ .Values.global.namespace | default "dephealth-conformance" }}
{{- end -}}
