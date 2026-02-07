{{/*
Полный путь к образу: registry/image:tag
Использование: {{ include "dephealth-infra.image" (dict "registry" .Values.global.imageRegistry "image" "postgres" "tag" "17-alpine") }}
*/}}
{{- define "dephealth-infra.image" -}}
{{- if .registry -}}
{{ .registry }}/{{ .image }}:{{ .tag }}
{{- else -}}
{{ .image }}:{{ .tag }}
{{- end -}}
{{- end -}}

{{/*
Путь к custom-образу (stubs): pushRegistry/image:tag или image:tag
Использование: {{ include "dephealth-infra.customImage" (dict "registry" .Values.global.pushRegistry "image" "dephealth-http-stub" "tag" "latest") }}
*/}}
{{- define "dephealth-infra.customImage" -}}
{{- if .registry -}}
{{ .registry }}/{{ .image }}:{{ .tag }}
{{- else -}}
{{ .image }}:{{ .tag }}
{{- end -}}
{{- end -}}

{{/*
Общие метки для всех ресурсов.
*/}}
{{- define "dephealth-infra.labels" -}}
app.kubernetes.io/part-of: dephealth
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
{{- end -}}

{{/*
Метки селектора для конкретного компонента.
Использование: {{ include "dephealth-infra.selectorLabels" (dict "name" "postgres-primary") }}
*/}}
{{- define "dephealth-infra.selectorLabels" -}}
app.kubernetes.io/name: {{ .name }}
app.kubernetes.io/part-of: dephealth
{{- end -}}

{{/*
Целевой namespace.
*/}}
{{- define "dephealth-infra.namespace" -}}
{{ .Values.global.namespace | default "dephealth-test" }}
{{- end -}}
