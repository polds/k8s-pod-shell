{{- define "kubeshell-web.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kubeshell-web.fullname" -}}
{{- printf "%s-%s" .Release.Name (include "kubeshell-web.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kubeshell-web.serviceAccountName" -}}
{{- include "kubeshell-web.fullname" . -}}
{{- end -}}
