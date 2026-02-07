{{/*
Путь к custom-образу: pushRegistry/image:tag или image:tag
*/}}
{{- define "dephealth-services.image" -}}
{{- if .registry -}}
{{ .registry }}/{{ .image }}:{{ .tag }}
{{- else -}}
{{ .image }}:{{ .tag }}
{{- end -}}
{{- end -}}

{{/*
Общие метки.
*/}}
{{- define "dephealth-services.labels" -}}
app.kubernetes.io/part-of: dephealth
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
{{- end -}}

{{/*
Метки селектора для конкретного сервиса.
*/}}
{{- define "dephealth-services.selectorLabels" -}}
app: {{ .name }}
app.kubernetes.io/name: {{ .name }}
app.kubernetes.io/part-of: dephealth
{{- end -}}

{{/*
Целевой namespace.
*/}}
{{- define "dephealth-services.namespace" -}}
{{ .Values.global.namespace | default "dephealth-test" }}
{{- end -}}

{{/*
FQDN хоста инфраструктурного сервиса.
Если infraNamespace задан и отличается от текущего — добавляет .namespace.svc.
*/}}
{{- define "dephealth-services.infraHost" -}}
{{- if and .infraNs (ne .infraNs .currentNs) -}}
{{ .host }}.{{ .infraNs }}.svc
{{- else -}}
{{ .host }}
{{- end -}}
{{- end -}}
