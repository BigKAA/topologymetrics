{{/*
Полный путь к образу: registry/image:tag
Использование: {{ include "dephealth-monitoring.image" (dict "registry" .Values.global.imageRegistry "image" "grafana/grafana" "tag" "11.6.0") }}
*/}}
{{- define "dephealth-monitoring.image" -}}
{{- if .registry -}}
{{ .registry }}/{{ .image }}:{{ .tag }}
{{- else -}}
{{ .image }}:{{ .tag }}
{{- end -}}
{{- end -}}

{{/*
Общие метки для всех ресурсов.
*/}}
{{- define "dephealth-monitoring.labels" -}}
app.kubernetes.io/part-of: dephealth
app.kubernetes.io/component: monitoring
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
{{- end -}}

{{/*
Метки селектора для конкретного компонента.
Использование: {{ include "dephealth-monitoring.selectorLabels" (dict "name" "victoriametrics") }}
*/}}
{{- define "dephealth-monitoring.selectorLabels" -}}
app: {{ .name }}
app.kubernetes.io/name: {{ .name }}
app.kubernetes.io/part-of: dephealth
{{- end -}}

{{/*
Целевой namespace.
*/}}
{{- define "dephealth-monitoring.namespace" -}}
{{ .Values.global.namespace | default "dephealth-monitoring" }}
{{- end -}}
