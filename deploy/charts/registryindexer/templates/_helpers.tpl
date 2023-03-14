{{/*
Expand the name of the chart.
*/}}
{{- define "registryindexer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "registryindexer.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "registryindexer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "registryindexer.labels" -}}
helm.sh/chart: {{ include "registryindexer.chart" . }}
{{ include "registryindexer.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "registryindexer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "registryindexer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "registryindexer.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "registryindexer.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}



{{/*
Create the name of the configMap to use
*/}}
{{- define "registryindexer.configMapName" -}}
{{- default (include "registryindexer.fullname" .) .Values.configMap.name }}
{{- end }}

{{/*
Create the name of the TLS secret to use
*/}}
{{- define "registryindexer.TLSSecretName" -}}
{{- if empty .Values.ingress.tls.name -}}
{{- printf "%s-tls" (include "registryindexer.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- .Values.ingress.tls.name -}}
{{- end }}
{{- end }}
