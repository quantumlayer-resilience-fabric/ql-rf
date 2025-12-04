{{/*
Expand the name of the chart.
*/}}
{{- define "ql-rf.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "ql-rf.fullname" -}}
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
{{- define "ql-rf.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "ql-rf.labels" -}}
helm.sh/chart: {{ include "ql-rf.chart" . }}
{{ include "ql-rf.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "ql-rf.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ql-rf.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "ql-rf.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "ql-rf.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
API service name
*/}}
{{- define "ql-rf.api.fullname" -}}
{{- printf "%s-api" (include "ql-rf.fullname" .) }}
{{- end }}

{{/*
Orchestrator service name
*/}}
{{- define "ql-rf.orchestrator.fullname" -}}
{{- printf "%s-orchestrator" (include "ql-rf.fullname" .) }}
{{- end }}

{{/*
Connectors service name
*/}}
{{- define "ql-rf.connectors.fullname" -}}
{{- printf "%s-connectors" (include "ql-rf.fullname" .) }}
{{- end }}

{{/*
Drift service name
*/}}
{{- define "ql-rf.drift.fullname" -}}
{{- printf "%s-drift" (include "ql-rf.fullname" .) }}
{{- end }}

{{/*
UI service name
*/}}
{{- define "ql-rf.ui.fullname" -}}
{{- printf "%s-ui" (include "ql-rf.fullname" .) }}
{{- end }}

{{/*
Database URL
*/}}
{{- define "ql-rf.databaseUrl" -}}
{{- if .Values.postgresql.enabled }}
{{- printf "postgres://%s:%s@%s-postgresql:5432/%s?sslmode=disable" .Values.postgresql.auth.username .Values.postgresql.auth.password (include "ql-rf.fullname" .) .Values.postgresql.auth.database }}
{{- else }}
{{- printf "postgres://%s@%s:%v/%s?sslmode=disable" .Values.externalDatabase.username .Values.externalDatabase.host .Values.externalDatabase.port .Values.externalDatabase.database }}
{{- end }}
{{- end }}

{{/*
Redis URL
*/}}
{{- define "ql-rf.redisUrl" -}}
{{- if .Values.redis.enabled }}
{{- printf "redis://:%s@%s-redis-master:6379" .Values.redis.auth.password (include "ql-rf.fullname" .) }}
{{- else }}
{{- printf "redis://%s:%v" .Values.externalRedis.host .Values.externalRedis.port }}
{{- end }}
{{- end }}

{{/*
Temporal address
*/}}
{{- define "ql-rf.temporalAddress" -}}
{{- if .Values.temporal.enabled }}
{{- printf "%s-temporal-frontend:7233" (include "ql-rf.fullname" .) }}
{{- else }}
{{- printf "%s:%v" .Values.externalTemporal.host .Values.externalTemporal.port }}
{{- end }}
{{- end }}

{{/*
Image name with tag
*/}}
{{- define "ql-rf.image" -}}
{{- $registry := .global.imageRegistry | default "" }}
{{- $repository := .image.repository }}
{{- $tag := .image.tag | default .appVersion }}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry $repository $tag }}
{{- else }}
{{- printf "%s:%s" $repository $tag }}
{{- end }}
{{- end }}

{{/*
Secret name for credentials
*/}}
{{- define "ql-rf.secretName" -}}
{{- printf "%s-secrets" (include "ql-rf.fullname" .) }}
{{- end }}
