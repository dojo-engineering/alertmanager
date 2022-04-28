// Copyright 2018 Prometheus Team
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package template

import (
	"net/url"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	"github.com/prometheus/alertmanager/types"
)

func TestPairNames(t *testing.T) {
	pairs := Pairs{
		{"name1", "value1"},
		{"name2", "value2"},
		{"name3", "value3"},
	}

	expected := []string{"name1", "name2", "name3"}
	require.EqualValues(t, expected, pairs.Names())
}

func TestPairValues(t *testing.T) {
	pairs := Pairs{
		{"name1", "value1"},
		{"name2", "value2"},
		{"name3", "value3"},
	}

	expected := []string{"value1", "value2", "value3"}
	require.EqualValues(t, expected, pairs.Values())
}

func TestKVSortedPairs(t *testing.T) {
	kv := KV{"d": "dVal", "b": "bVal", "c": "cVal"}

	expectedPairs := Pairs{
		{"b", "bVal"},
		{"c", "cVal"},
		{"d", "dVal"},
	}

	for i, p := range kv.SortedPairs() {
		require.EqualValues(t, p.Name, expectedPairs[i].Name)
		require.EqualValues(t, p.Value, expectedPairs[i].Value)
	}

	// validates alertname always comes first
	kv = KV{"d": "dVal", "b": "bVal", "c": "cVal", "alertname": "alert", "a": "aVal"}

	expectedPairs = Pairs{
		{"alertname", "alert"},
		{"a", "aVal"},
		{"b", "bVal"},
		{"c", "cVal"},
		{"d", "dVal"},
	}

	for i, p := range kv.SortedPairs() {
		require.EqualValues(t, p.Name, expectedPairs[i].Name)
		require.EqualValues(t, p.Value, expectedPairs[i].Value)
	}
}

func TestKVRemove(t *testing.T) {
	kv := KV{
		"key1": "val1",
		"key2": "val2",
		"key3": "val3",
		"key4": "val4",
	}

	kv = kv.Remove([]string{"key2", "key4"})

	expected := []string{"key1", "key3"}
	require.EqualValues(t, expected, kv.Names())
}

func TestAlertsFiring(t *testing.T) {
	alerts := Alerts{
		{Status: string(model.AlertFiring)},
		{Status: string(model.AlertResolved)},
		{Status: string(model.AlertFiring)},
		{Status: string(model.AlertResolved)},
		{Status: string(model.AlertResolved)},
	}

	for _, alert := range alerts.Firing() {
		if alert.Status != string(model.AlertFiring) {
			t.Errorf("unexpected status %q", alert.Status)
		}
	}
}

func TestAlertsResolved(t *testing.T) {
	alerts := Alerts{
		{Status: string(model.AlertFiring)},
		{Status: string(model.AlertResolved)},
		{Status: string(model.AlertFiring)},
		{Status: string(model.AlertResolved)},
		{Status: string(model.AlertResolved)},
	}

	for _, alert := range alerts.Resolved() {
		if alert.Status != string(model.AlertResolved) {
			t.Errorf("unexpected status %q", alert.Status)
		}
	}
}

func TestData(t *testing.T) {
	u, err := url.Parse("http://example.com/")
	require.NoError(t, err)
	tmpl := &Template{ExternalURL: u}
	startTime := time.Time{}.Add(1 * time.Second)
	endTime := time.Time{}.Add(2 * time.Second)

	for _, tc := range []struct {
		receiver    string
		groupLabels model.LabelSet
		alerts      []*types.Alert

		exp *Data
	}{
		{
			receiver: "webhook",
			exp: &Data{
				Receiver:          "webhook",
				Status:            "resolved",
				Alerts:            Alerts{},
				GroupLabels:       KV{},
				CommonLabels:      KV{},
				CommonAnnotations: KV{},
				ExternalURL:       u.String(),
			},
		},
		{
			receiver: "webhook",
			groupLabels: model.LabelSet{
				model.LabelName("job"): model.LabelValue("foo"),
			},
			alerts: []*types.Alert{
				{
					Alert: model.Alert{
						StartsAt: startTime,
						Labels: model.LabelSet{
							model.LabelName("severity"): model.LabelValue("warning"),
							model.LabelName("job"):      model.LabelValue("foo"),
						},
						Annotations: model.LabelSet{
							model.LabelName("description"): model.LabelValue("something happened"),
							model.LabelName("runbook"):     model.LabelValue("foo"),
						},
					},
				},
				{
					Alert: model.Alert{
						StartsAt: startTime,
						EndsAt:   endTime,
						Labels: model.LabelSet{
							model.LabelName("severity"): model.LabelValue("critical"),
							model.LabelName("job"):      model.LabelValue("foo"),
						},
						Annotations: model.LabelSet{
							model.LabelName("description"): model.LabelValue("something else happened"),
							model.LabelName("runbook"):     model.LabelValue("foo"),
						},
					},
				},
			},
			exp: &Data{
				Receiver: "webhook",
				Status:   "firing",
				Alerts: Alerts{
					{
						Status:      "firing",
						Labels:      KV{"severity": "warning", "job": "foo"},
						Annotations: KV{"description": "something happened", "runbook": "foo"},
						StartsAt:    startTime,
						Fingerprint: "9266ef3da838ad95",
					},
					{
						Status:      "resolved",
						Labels:      KV{"severity": "critical", "job": "foo"},
						Annotations: KV{"description": "something else happened", "runbook": "foo"},
						StartsAt:    startTime,
						EndsAt:      endTime,
						Fingerprint: "3b15fd163d36582e",
					},
				},
				GroupLabels:       KV{"job": "foo"},
				CommonLabels:      KV{"job": "foo"},
				CommonAnnotations: KV{"runbook": "foo"},
				ExternalURL:       u.String(),
			},
		},
		{
			receiver:    "webhook",
			groupLabels: model.LabelSet{},
			alerts: []*types.Alert{
				{
					Alert: model.Alert{
						StartsAt: startTime,
						Labels: model.LabelSet{
							model.LabelName("severity"): model.LabelValue("warning"),
							model.LabelName("job"):      model.LabelValue("foo"),
						},
						Annotations: model.LabelSet{
							model.LabelName("description"): model.LabelValue("something happened"),
							model.LabelName("runbook"):     model.LabelValue("foo"),
						},
					},
				},
				{
					Alert: model.Alert{
						StartsAt: startTime,
						EndsAt:   endTime,
						Labels: model.LabelSet{
							model.LabelName("severity"): model.LabelValue("critical"),
							model.LabelName("job"):      model.LabelValue("bar"),
						},
						Annotations: model.LabelSet{
							model.LabelName("description"): model.LabelValue("something else happened"),
							model.LabelName("runbook"):     model.LabelValue("bar"),
						},
					},
				},
			},
			exp: &Data{
				Receiver: "webhook",
				Status:   "firing",
				Alerts: Alerts{
					{
						Status:      "firing",
						Labels:      KV{"severity": "warning", "job": "foo"},
						Annotations: KV{"description": "something happened", "runbook": "foo"},
						StartsAt:    startTime,
						Fingerprint: "9266ef3da838ad95",
					},
					{
						Status:      "resolved",
						Labels:      KV{"severity": "critical", "job": "bar"},
						Annotations: KV{"description": "something else happened", "runbook": "bar"},
						StartsAt:    startTime,
						EndsAt:      endTime,
						Fingerprint: "c7e68cb08e3e67f9",
					},
				},
				GroupLabels:       KV{},
				CommonLabels:      KV{},
				CommonAnnotations: KV{},
				ExternalURL:       u.String(),
			},
		},
	} {
		tc := tc
		t.Run("", func(t *testing.T) {
			got := tmpl.Data(tc.receiver, tc.groupLabels, tc.alerts...)
			require.Equal(t, tc.exp, got)
		})
	}
}

func TestTemplateExpansion(t *testing.T) {
	tmpl, err := FromGlobs()
	require.NoError(t, err)

	for _, tc := range []struct {
		title string
		in    string
		data  interface{}
		html  bool

		exp  string
		fail bool
	}{
		{
			title: "Template without action",
			in:    `abc`,
			exp:   "abc",
		},
		{
			title: "Template with simple action",
			in:    `{{ "abc" }}`,
			exp:   "abc",
		},
		{
			title: "Template with invalid syntax",
			in:    `{{ `,
			fail:  true,
		},
		{
			title: "Template using toUpper",
			in:    `{{ "abc" | toUpper }}`,
			exp:   "ABC",
		},
		{
			title: "Template using toLower",
			in:    `{{ "ABC" | toLower }}`,
			exp:   "abc",
		},
		{
			title: "Template using title",
			in:    `{{ "abc" | title }}`,
			exp:   "Abc",
		},
		{
			title: "Template using positive match",
			in:    `{{ if match "^a" "abc"}}abc{{ end }}`,
			exp:   "abc",
		},
		{
			title: "Template using negative match",
			in:    `{{ if match "abcd" "abc" }}abc{{ end }}`,
			exp:   "",
		},
		{
			title: "Template using join",
			in:    `{{ . | join "," }}`,
			data:  []string{"a", "b", "c"},
			exp:   "a,b,c",
		},
		{
			title: "Text template without HTML escaping",
			in:    `{{ "<b>" }}`,
			exp:   "<b>",
		},
		{
			title: "HTML template with escaping",
			in:    `{{ "<b>" }}`,
			html:  true,
			exp:   "&lt;b&gt;",
		},
		{
			title: "HTML template using safeHTML",
			in:    `{{ "<b>" | safeHtml }}`,
			html:  true,
			exp:   "<b>",
		},
		{
			title: "Template using reReplaceAll",
			in:    `{{ reReplaceAll "ab" "AB" "abcdabcda"}}`,
			exp:   "ABcdABcda",
		},
		{
			title: "Template using stringSlice",
			in:    `{{ with .GroupLabels }}{{ with .Remove (stringSlice "key1" "key3") }}{{ .SortedPairs.Values }}{{ end }}{{ end }}`,
			data: Data{
				GroupLabels: KV{
					"key1": "key1",
					"key2": "key2",
					"key3": "key3",
					"key4": "key4",
				},
			},
			exp: "[key2 key4]",
		},
	} {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			f := tmpl.ExecuteTextString
			if tc.html {
				f = tmpl.ExecuteHTMLString
			}
			got, err := f(tc.in, tc.data)
			if tc.fail {
				require.NotNil(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.exp, got)
		})
	}
}

func TestDojoTemplates(t *testing.T) {
	tmpl, err := FromGlobs()
	require.NoError(t, err)

	commonTmpl := `{{- /* */ -}}
{{- /* dojo.subject.stable_text(Data) */ -}}
{{- /* This provides a stable subject value that is only a function of the static */ -}}
{{- /* values used to group the alerts being notified (GroupLabels). */ -}}
{{- /* It does NOT make use of the number of alerts, common labels across firing */ -}}
{{- /* alerts (CommonLabels) or annotations (CommonAnnotations). All these values */ -}}
{{- /* can change as new alerts fire and old alerts resolve, making the subject */ -}}
{{- /* value change as well. */ -}}
{{- /* Having a stable subject is good for use cases where the grouped alerts state */ -}}
{{- /* is to me synchronised with another system (eg: PagerDuty), and we */ -}}
{{- /* want to prevent a stale subject from misleading people. */ -}}
{{- /* Examples: */ -}}
{{- /* "Alerts for $receiver" */ -}}
{{- /* "[$alertname_value] ($group_label_key=$group_label_value)" */ -}}
{{- /* "($group_label_key=$group_label_value)" */ -}}
{{- define "dojo.subject.stable_text" -}}
	{{- if eq .Status "resolved" -}}
		{{- "Resolved: " -}}
	{{- end -}}
	{{- with .GroupLabels -}}
		{{- $groupLabels := . -}}
		{{- with index . "alertname" -}}
			{{- "[" }}{{ . }}{{ "]" -}}
		{{- end -}}
		{{- with .Remove (stringSlice "alertname") -}}
			{{- with index $groupLabels "alertname" -}}
				{{- " " -}}
			{{- end -}}
			{{ "(" }}
			{{- with index .SortedPairs 0 -}}
				{{- .Name }}={{ .Value -}}
			{{- end -}}
			{{- with slice .SortedPairs 1 -}}
				{{- range . -}}
					{{- " " }}{{ .Name }}={{ .Value -}}
				{{- end -}}
			{{- end -}}
			{{ ")" }}
		{{- end -}}
	{{- else -}}
		{{- "Alerts for " -}}{{- .Receiver -}}
	{{- end -}}
{{- end -}}

{{- /* dojo.alert.text(Alert) */ -}}
{{- /* Textual representation of the Alert with: */ -}}
{{- /* - A descriptive "alert name": */ -}}
{{- /*   - "[$alertname_value] ($label=$value)" when "alertname" label exists. */ -}}
{{- /*   - "($label=$value)" when "alertname" is missing. */ -}}
{{- /* - Annotations */ -}}
{{- define "dojo.alert.text" -}}
	{{- $alert := . -}}
	{{- with index .Labels.alertname -}}
		{{- "[" -}}{{ . }}{{- "]" -}}
		{{- with ($alert.Labels.Remove (stringSlice "alertname")).SortedPairs -}}
			{{- " (" }}
			{{- with index . 0 -}}
				{{- .Name }}={{ .Value -}}
			{{- end -}}
			{{- range slice . 1 -}}
				{{- " " }}{{ .Name }}={{ .Value -}}
			{{- end -}}
			{{- ")" }}
		{{- end -}}
	{{- else -}}
		{{- "(" -}}
			{{- with index $alert.Labels.SortedPairs 0 -}}
				{{- .Name }}={{ .Value -}}
			{{- end -}}
			{{- with slice $alert.Labels.SortedPairs 1 -}}
				{{- range . -}}
					{{- " " }}{{ .Name }}={{ .Value -}}
				{{- end -}}
			{{- end -}}
		{{- ")" -}}
	{{- end -}}
	{{- with .Annotations.SortedPairs -}}
		{{- "\nAnnotations:" -}}
		{{- range . -}}
			{{ "\n" }}{{ .Name }}: {{ .Value }}
		{{- end -}}
	{{- end -}}
	{{- with .GeneratorURL -}}
		{{ "\n" }}{{ . }}
	{{- end -}}
{{- end -}}

{{- /* dojo.alerts.text(Alerts) */ -}}
{{- /* Textual representation of a list of Alerts. */ -}}
{{- /* See also: dojo.alert.text */ -}}
{{- /* See also: dojo.alerts.status_grouped_text */ -}}
{{- define "dojo.alerts.text" -}}
	{{- with . -}}
		{{- with index . 0 -}}
			{{- template "dojo.alert.text" . -}}
		{{- end -}}
		{{- with slice . 1 -}}
			{{- range . -}}
				{{- "\n\n" }}
				{{- template "dojo.alert.text" . -}}
			{{- end -}}
		{{- end -}}
	{{- end -}}
{{- end -}}

{{- /* dojo.alerts.status_grouped_text(Alerts) */ -}}
{{- /* Textual representation of a list of Alerts with status */ -}}
{{- /* (firing / resolved) grouping. */ -}}
{{- /* See also: dojo.alert.text */ -}}
{{- /* See also: dojo.alerts.text */ -}}
{{- define "dojo.alerts.status_grouped_text" -}}
	{{- if and (gt (len .Firing) 0) (gt (len .Resolved) 0)  }}
		{{- $alerts := . -}}
		{{- with .Firing -}}
			{{- "FIRING:\n\n" -}}
			{{- template "dojo.alerts.text" . -}}
		{{- end -}}
		{{- with .Resolved -}}
			{{- with $alerts.Firing -}}
				{{- "\n\n" -}}
			{{- end -}}
			{{- "RESOLVED:\n\n" -}}
			{{- template "dojo.alerts.text" . -}}
		{{- end -}}
	{{- else -}}
		{{- template "dojo.alerts.text" .Firing -}}
		{{- template "dojo.alerts.text" .Resolved -}}
	{{- end -}}
{{- end -}}

{{- /* dojo.alerts.url.firing(Data) */ -}}
{{- /* URL that points to Grafana Labs list of firing alerts. */ -}}
{{- define "dojo.alerts.url.firing" -}}
	{{- "https://paymentsense.grafana.net/alerting/list?" }}
		{{- "dataSource=DATASOURCE_NAME" -}}
		{{- "&queryString=" -}}
			{{- /* tenant is a mandatory field for all configured alerts, */ -}}
			{{- /* validated by CI so it is guaranteed to exist at CommonLabels */ -}}
			{{- "tenant%3D" -}}{{- .CommonLabels.tenant | urlquery -}}{{- "," -}}
			{{- /* urgency is to become mandatory via CI... */ -}}
			{{- with .CommonLabels.urgency -}}
				{{- "urgency%3D" -}}{{- . | urlquery -}}{{- "," -}}
			{{- /* ...until then, we must account for when it is missing. */ -}}
			{{- else -}}
				{{- "urgency!~%5E(high%7Clow)$," -}}
			{{- end -}}
			{{- with .GroupLabels.Remove (stringSlice "tenant" "urgency") -}}
				{{- range .SortedPairs -}}
					{{- .Name | urlquery -}}{{- "%3D" -}}{{- .Value | urlquery -}}{{- "," -}}
				{{- end -}}
			{{- end -}}
		{{- "&ruleType=alerting" -}}
		{{- "&alertState=firing" -}}
{{- end -}}

{{- /* dojo.alerts.url.history(Data) */ -}}
{{- /* URL that points to a Grafana Labs Dashboard with the history of alerts */ -}}
{{- define "dojo.alerts.url.history" -}}
	{{- "https://paymentsense.grafana.net/d/luyBQ9Y7z/?" -}}
		{{- "orgId=1&" -}}
		{{- "var-data_source=DATASOURCE_NAME&" -}}
		{{- /* tenant is a mandatory field for all configured alerts, */ -}}
		{{- /* validated by CI so it is guaranteed to exist at CommonLabels */ -}}
		{{- "var-tenant=" -}}{{- .CommonLabels.tenant | urlquery -}}{{- "&" -}}
		{{- /* urgency is to become mandatory via CI... */ -}}
		{{- with .CommonLabels.urgency -}}
			{{- "var-urgency=" -}}{{- . | urlquery -}}{{- "&" -}}
		{{- /* ...until then, we must account for when it is missing. */ -}}
		{{- else -}}
			{{- "var-label=urgency%7C!~%7C%5E(high__gfp__low)$" -}}{{- "&" -}}
		{{- end -}}
		{{- /* alertname MAY be part of grouped labels */ -}}
		{{- with .GroupLabels.alertname -}}
			{{- "var-alertname=" -}}{{- . | urlquery -}}{{- "&" -}}
		{{- end -}}
		{{- /* Add any other group labels other than the ones already set */ -}}
		{{- with .GroupLabels.Remove (stringSlice "tenant" "urgency" "alertname") -}}
			{{- range .SortedPairs -}}
				{{- "var-label=" -}}
					{{- .Name | urlquery -}}
					{{- "%7C%3D%7C" -}}
					{{- .Value | urlquery -}}
					{{- "&" -}}
			{{- end -}}
		{{- end -}}
{{- end -}}

{{- /* dojo.alerts.url.new_silence(Data) */ -}}
{{- /* URL that points to a Grafana Labs page where a silence to all matching */ -}}
{{- /* alerts can be added */ -}}
{{- define "dojo.alerts.url.new_silence" -}}
	{{- "https://paymentsense.grafana.net/alerting/silence/new?" -}}
		{{- "alertmanager=ALERTMANAGER_NAME&" -}}
		{{- /* tenant is a mandatory field for all configured alerts, */ -}}
		{{- /* validated by CI so it is guaranteed to exist at CommonLabels */ -}}
		{{- "matcher=tenant%3D" -}}{{- .CommonLabels.tenant | urlquery -}}{{- "&" -}}
		{{- /* urgency is to become mandatory via CI... */ -}}
		{{- with .CommonLabels.urgency -}}
			{{- "matcher=urgency%3D" -}}{{- . | urlquery -}}{{- "&" -}}
		{{- /* ...until then, we must account for when it is missing. */ -}}
		{{- else -}}
			{{- "matcher=urgency!~%5E(high|low)$" -}}{{- "&" -}}
		{{- end -}}
		{{- /* Add any other group labels other than the ones already set */ -}}
		{{- with .GroupLabels.Remove (stringSlice "tenant" "urgency") -}}
			{{- range .SortedPairs -}}
				{{- "matcher=" -}}
					{{- .Name | urlquery -}}
					{{- "%3D" -}}
					{{- .Value | urlquery -}}
					{{- "&" -}}
			{{- end -}}
		{{- end -}}
{{- end -}}

{{- /* dojo.documentation.links(Data) */ -}}
{{- /* Docummentation with links to references regarding the notification */ -}}
{{- define "dojo.documentation.links" -}}
	{{- "Below are links referring to all alerts grouped. You must work until all of them are resolved.\n" -}}
	{{- "\n" -}}
	{{- "Currently firing alerts for this incident:\n" -}}
	{{ template "dojo.alerts.url.firing" . }}{{- "\n" -}}
	{{- "\n" -}}
	{{- "History of alerts for this incident:\n" -}}
	{{ template "dojo.alerts.url.history" . }}
{{- end -}}

{{- /* dojo.documentation.high_urgency(Data) */ -}}
{{- /* Docummentation to be used in notifications for high urgency alerts */ -}}
{{- define "dojo.documentation.high_urgency" -}}
	{{- "Alert(s) of high urgency have fired meaning there's likely business impact going on. Please work on fixing the problem IMMEDIATELY!\n" -}}
	{{- "\n" -}}
	{{- template "dojo.documentation.links" . -}}
{{- end -}}

{{- /* dojo.documentation.low_urgency(Data) */ -}}
{{- /* Docummentation to be used in notifications for low urgency alerts */ -}}
{{- define "dojo.documentation.low_urgency" -}}
	{{- "Alert(s) of low urgency have fired. They indicate that there's either tolerable business impact or potential business impact if no action is taken within the next business day. Evaluate the firing alerts and take necessary actions to fix / prevent any problems.\n" -}}
	{{- "\n" -}}
	{{- "If you identify that no action is required, it means the alert misfired, meaning the action to be taken here is to adjust the alert tirggering mechanism so that it only fires when it is actionable. If you find the alert signal useful, despite not being actionable, then the signal can be moved to a dashboard.\n" -}}
	{{- "\n" -}}
	{{- template "dojo.documentation.links" . -}}
{{- end -}}

{{- /* dojo.documentation.unknown_urgency(Data) */ -}}
{{- /* Docummentation to be used in notifications for low urgency alerts */ -}}
{{- define "dojo.documentation.unknown_urgency" -}}
	{{- "An alert without a label urgency set to either high or low fired. The worst is assumed here: that the alert is of high urgency.\n" -}}
	{{- "\n" -}}
	{{- "This only happens when a misconfiguration happened on this alert, so it needs fixing. There are two actions required.\n" -}}
	{{- "\n" -}}
	{{- "The immediate action, is to evaluate the real urgency of the firing alert(s) and work on it accordingly.\n" -}}
	{{- "\n" -}}
	{{- "The secondary action, is to fix the alert configuration so that it fires with a correctly defined urgency next time.\n" -}}
	{{- "\n" -}}
	{{- template "dojo.documentation.links" . -}}
{{- end -}}
`
	for _, tc := range []struct {
		title string
		in    string
		data  interface{}

		exp  string
		fail bool
	}{
		// dojo.subject.stable_text
		{
			title: "dojo.subject.stable_text with no group_by and firing alerts",
			in:    `{{ template "dojo.subject.stable_text" . }}`,
			data: Data{
				Receiver: "Receiver",
				Status:   "firing",
			},
			exp: "Alerts for Receiver",
		},
		{
			title: "dojo.subject.stable_text with no group_by and resolved alerts",
			in:    `{{ template "dojo.subject.stable_text" . }}`,
			data: Data{
				Receiver: "Receiver",
				Status:   "resolved",
			},
			exp: "Resolved: Alerts for Receiver",
		},
		{
			title: "dojo.subject.stable_text with group_by, alertname and firing alerts",
			in:    `{{ template "dojo.subject.stable_text" . }}`,
			data: Data{
				Receiver: "Receiver",
				Status:   "firing",
				GroupLabels: KV{
					"alertname": "AlertName",
					"label1":    "value1",
					"label2":    "value2",
				},
			},
			exp: "[AlertName] (label1=value1 label2=value2)",
		},
		{
			title: "dojo.subject.stable_text with group_by, alertname and resolved alerts",
			in:    `{{ template "dojo.subject.stable_text" . }}`,
			data: Data{
				Receiver: "Receiver",
				Status:   "resolved",
				GroupLabels: KV{
					"alertname": "AlertName",
					"label1":    "value1",
					"label2":    "value2",
				},
			},
			exp: "Resolved: [AlertName] (label1=value1 label2=value2)",
		},
		{
			title: "dojo.subject.stable_text with group_by and firing alerts",
			in:    `{{ template "dojo.subject.stable_text" . }}`,
			data: Data{
				Receiver: "Receiver",
				Status:   "firing",
				GroupLabels: KV{
					"label1": "value1",
					"label2": "value2",
				},
			},
			exp: "(label1=value1 label2=value2)",
		},
		{
			title: "dojo.subject.stable_text with group_by and resolved alerts",
			in:    `{{ template "dojo.subject.stable_text" . }}`,
			data: Data{
				Receiver: "Receiver",
				Status:   "resolved",
				GroupLabels: KV{
					"label1": "value1",
					"label2": "value2",
				},
			},
			exp: "Resolved: (label1=value1 label2=value2)",
		},
		// dojo.alert.text
		{
			title: "dojo.alert.text with alertname and labels",
			in:    `{{ template "dojo.alert.text" (index .Alerts 0) }}`,
			data: Data{
				Alerts: Alerts{
					{
						Status: "firing",
						Labels: KV{
							"alertname": "AlertName",
							"label1":    "value1",
							"label2":    "value2",
						},
						Annotations: KV{
							"annotation1": "value1",
							"annotation2": "value2",
						},
						GeneratorURL: "http://generator.url/",
					},
				},
			},
			exp: "[AlertName] (label1=value1 label2=value2)\n" +
				"Annotations:\n" +
				"annotation1: value1\n" +
				"annotation2: value2\n" +
				"http://generator.url/",
		},
		{
			title: "dojo.alert.text with alertname and no extra labels",
			in:    `{{ template "dojo.alert.text" (index .Alerts 0) }}`,
			data: Data{
				Alerts: Alerts{
					{
						Status: "firing",
						Labels: KV{
							"alertname": "AlertName",
						},
						Annotations: KV{
							"annotation1": "value1",
							"annotation2": "value2",
						},
						GeneratorURL: "http://generator.url/",
					},
				},
			},
			exp: "[AlertName]\n" +
				"Annotations:\n" +
				"annotation1: value1\n" +
				"annotation2: value2\n" +
				"http://generator.url/",
		},
		{
			title: "dojo.alert.text without alertname and labels",
			in:    `{{ template "dojo.alert.text" (index .Alerts 0) }}`,
			data: Data{
				Alerts: Alerts{
					{
						Status: "firing",
						Labels: KV{
							"label1": "value1",
							"label2": "value2",
						},
						Annotations: KV{
							"annotation1": "value1",
							"annotation2": "value2",
						},
						GeneratorURL: "http://generator.url/",
					},
				},
			},
			exp: "(label1=value1 label2=value2)\n" +
				"Annotations:\n" +
				"annotation1: value1\n" +
				"annotation2: value2\n" +
				"http://generator.url/",
		},
		// dojo.alerts.status_grouped_text
		{
			title: "dojo.alerts.status_grouped_text with firing & resolved",
			in:    `{{ template "dojo.alerts.status_grouped_text" .Alerts }}`,
			data: Data{
				Alerts: Alerts{
					{
						Status: "firing",
						Labels: KV{
							"alertname": "AlertName1",
							"label1":    "value1",
							"label2":    "value2",
						},
						Annotations: KV{
							"annotation1": "value1",
							"annotation2": "value2",
						},
						GeneratorURL: "http://generator.url/",
					},
					{
						Status: "resolved",
						Labels: KV{
							"label1": "value1",
							"label2": "value2",
						},
						Annotations: KV{
							"annotation1": "value1",
							"annotation2": "value2",
						},
						GeneratorURL: "http://generator.url/",
					},
				},
			},
			exp: "FIRING:\n" +
				"\n" +
				"[AlertName1] (label1=value1 label2=value2)\n" +
				"Annotations:\n" +
				"annotation1: value1\n" +
				"annotation2: value2\n" +
				"http://generator.url/\n" +
				"\n" +
				"RESOLVED:\n" +
				"\n" +
				"(label1=value1 label2=value2)\n" +
				"Annotations:\n" +
				"annotation1: value1\n" +
				"annotation2: value2\n" +
				"http://generator.url/",
		},
		{
			title: "dojo.alerts.status_grouped_text with only firing",
			in:    `{{ template "dojo.alerts.status_grouped_text" .Alerts }}`,
			data: Data{
				Alerts: Alerts{
					{
						Status: "firing",
						Labels: KV{
							"alertname": "AlertName1",
							"label1":    "value1",
							"label2":    "value2",
						},
						Annotations: KV{
							"annotation1": "value1",
							"annotation2": "value2",
						},
						GeneratorURL: "http://generator.url/",
					},
					{
						Status: "firing",
						Labels: KV{
							"label1": "value1",
							"label2": "value2",
						},
						Annotations: KV{
							"annotation1": "value1",
							"annotation2": "value2",
						},
						GeneratorURL: "http://generator.url/",
					},
				},
			},
			exp: "[AlertName1] (label1=value1 label2=value2)\n" +
				"Annotations:\n" +
				"annotation1: value1\n" +
				"annotation2: value2\n" +
				"http://generator.url/\n" +
				"\n" +
				"(label1=value1 label2=value2)\n" +
				"Annotations:\n" +
				"annotation1: value1\n" +
				"annotation2: value2\n" +
				"http://generator.url/",
		},
		// dojo.alerts.url.firing
		{
			title: "dojo.alerts.url.firing with group_by and with alertname and urgency",
			in:    `{{ template "dojo.alerts.url.firing" . }}`,
			data: Data{
				GroupLabels: KV{
					"alertname": "AlertName",
					"foo":       "$b a r",
				},
				CommonLabels: KV{
					"tenant":    "example",
					"urgency":   "high",
					"alertname": "AlertName",
					"must":      "not",
					"be":        "used",
				},
			},
			exp: "https://paymentsense.grafana.net/alerting/list?" +
				"dataSource=DATASOURCE_NAME&" +
				"queryString=" +
				"tenant%3Dexample," +
				"urgency%3Dhigh," +
				"alertname%3DAlertName," +
				"foo%3D%24b+a+r,&" +
				"ruleType=alerting&" +
				"alertState=firing",
		},
		{
			title: "dojo.alerts.url.firing with group_by and without alertname and urgency",
			in:    `{{ template "dojo.alerts.url.firing" . }}`,
			data: Data{
				GroupLabels: KV{
					"foo": "$b a r",
				},
				CommonLabels: KV{
					"tenant":    "example",
					"alertname": "AlertName", // NOT to be used!
					"must":      "not",
					"be":        "used",
				},
			},
			exp: "https://paymentsense.grafana.net/alerting/list?" +
				"dataSource=DATASOURCE_NAME&" +
				"queryString=" +
				"tenant%3Dexample," +
				"urgency!~%5E(high%7Clow)$," +
				"foo%3D%24b+a+r,&" +
				"ruleType=alerting&" +
				"alertState=firing",
		},
		{
			title: "dojo.alerts.url.firing without group_by",
			in:    `{{ template "dojo.alerts.url.firing" . }}`,
			data: Data{
				CommonLabels: KV{
					"tenant":    "example",
					"urgency":   "high",
					"alertname": "AlertName",
					"must":      "not",
					"be":        "used",
				},
			},
			exp: "https://paymentsense.grafana.net/alerting/list?" +
				"dataSource=DATASOURCE_NAME&" +
				"queryString=" +
				"tenant%3Dexample," +
				"urgency%3Dhigh,&" +
				"ruleType=alerting&" +
				"alertState=firing",
		},
		// dojo.alerts.url.history
		{
			title: "dojo.alerts.url.history with group_by and with alertname and urgency",
			in:    `{{ template "dojo.alerts.url.history" . }}`,
			data: Data{
				GroupLabels: KV{
					"alertname": "AlertName",
					"foo":       "$b a r",
				},
				CommonLabels: KV{
					"tenant":    "example",
					"urgency":   "high",
					"alertname": "AlertName",
					"must":      "not",
					"be":        "used",
				},
			},
			exp: "https://paymentsense.grafana.net/d/luyBQ9Y7z/?orgId=1&" +
				"var-data_source=DATASOURCE_NAME&" +
				"var-tenant=example&" +
				"var-urgency=high&" +
				"var-alertname=AlertName&" +
				"var-label=foo%7C%3D%7C%24b+a+r&",
		},
		{
			title: "dojo.alerts.url.history with group_by and without alertname and urgency",
			in:    `{{ template "dojo.alerts.url.history" . }}`,
			data: Data{
				GroupLabels: KV{
					"foo": "$b a r",
				},
				CommonLabels: KV{
					"tenant":    "example",
					"alertname": "AlertName", // NOT to be used!
					"must":      "not",
					"be":        "used",
				},
			},
			exp: "https://paymentsense.grafana.net/d/luyBQ9Y7z/?orgId=1&" +
				"var-data_source=DATASOURCE_NAME&" +
				"var-tenant=example&" +
				"var-label=urgency%7C!~%7C%5E(high__gfp__low)$&" +
				"var-label=foo%7C%3D%7C%24b+a+r&",
		},
		{
			title: "dojo.alerts.url.history without group_by",
			in:    `{{ template "dojo.alerts.url.history" . }}`,
			data: Data{
				CommonLabels: KV{
					"tenant":    "example",
					"urgency":   "high",
					"alertname": "AlertName",
					"must":      "not",
					"be":        "used",
				},
			},
			exp: "https://paymentsense.grafana.net/d/luyBQ9Y7z/?orgId=1&" +
				"var-data_source=DATASOURCE_NAME&" +
				"var-tenant=example&" +
				"var-urgency=high&",
		},
		// dojo.alerts.url.new_silence
		{
			title: "dojo.alerts.url.new_silence with group_by and with alertname and urgency",
			in:    `{{ template "dojo.alerts.url.new_silence" . }}`,
			data: Data{
				GroupLabels: KV{
					"alertname": "AlertName",
					"foo":       "$b a r",
				},
				CommonLabels: KV{
					"tenant":    "example",
					"urgency":   "high",
					"alertname": "AlertName",
					"must":      "not",
					"be":        "used",
				},
			},
			exp: "https://paymentsense.grafana.net/alerting/silence/new?" +
				"alertmanager=ALERTMANAGER_NAME&" +
				"matcher=tenant%3Dexample&" +
				"matcher=urgency%3Dhigh&" +
				"matcher=alertname%3DAlertName&" +
				"matcher=foo%3D%24b+a+r&",
		},
		{
			title: "dojo.alerts.url.new_silence with group_by and without alertname and urgency",
			in:    `{{ template "dojo.alerts.url.new_silence" . }}`,
			data: Data{
				GroupLabels: KV{
					"foo": "$b a r",
				},
				CommonLabels: KV{
					"tenant":    "example",
					"alertname": "AlertName", // NOT to be used!
					"must":      "not",
					"be":        "used",
				},
			},
			exp: "https://paymentsense.grafana.net/alerting/silence/new?" +
				"alertmanager=ALERTMANAGER_NAME&" +
				"matcher=tenant%3Dexample&" +
				"matcher=urgency!~%5E(high|low)$&" +
				"matcher=foo%3D%24b+a+r&",
		},
		{
			title: "dojo.alerts.url.new_silence without group_by",
			in:    `{{ template "dojo.alerts.url.new_silence" . }}`,
			data: Data{
				CommonLabels: KV{
					"tenant":    "example",
					"urgency":   "high",
					"alertname": "AlertName",
					"must":      "not",
					"be":        "used",
				},
			},
			exp: "https://paymentsense.grafana.net/alerting/silence/new?" +
				"alertmanager=ALERTMANAGER_NAME&" +
				"matcher=tenant%3Dexample&" +
				"matcher=urgency%3Dhigh&",
		},
		// dojo.documentation.high_urgency
		{
			title: "dojo.documentation.high_urgency",
			in:    `{{ template "dojo.documentation.high_urgency" . }}`,
			data: Data{
				GroupLabels: KV{
					"label1": "value $1",
					"label2": "value $2",
				},
				CommonLabels: KV{
					"tenant":  "example",
					"urgency": "high",
				},
			},
			exp: "Alert(s) of high urgency have fired meaning there's likely business impact going on. Please work on fixing the problem IMMEDIATELY!\n" +
				"\n" +
				"Below are links referring to all alerts grouped. You must work until all of them are resolved.\n" +
				"\n" +
				"Currently firing alerts for this incident:\n" +
				"https://paymentsense.grafana.net/alerting/list?dataSource=DATASOURCE_NAME&queryString=tenant%3Dexample,urgency%3Dhigh,label1%3Dvalue+%241,label2%3Dvalue+%242,&ruleType=alerting&alertState=firing\n" +
				"\n" +
				"History of alerts for this incident:\n" +
				"https://paymentsense.grafana.net/d/luyBQ9Y7z/?orgId=1&var-data_source=DATASOURCE_NAME&var-tenant=example&var-urgency=high&var-label=label1%7C%3D%7Cvalue+%241&var-label=label2%7C%3D%7Cvalue+%242&",
		},
		// dojo.documentation.low_urgency
		{
			title: "dojo.documentation.low_urgency",
			in:    `{{ template "dojo.documentation.low_urgency" . }}`,
			data: Data{
				GroupLabels: KV{
					"label1": "value $1",
					"label2": "value $2",
				},
				CommonLabels: KV{
					"tenant":  "example",
					"urgency": "low",
				},
			},
			exp: "Alert(s) of low urgency have fired. They indicate that there's either tolerable business impact or potential business impact if no action is taken within the next business day. Evaluate the firing alerts and take necessary actions to fix / prevent any problems.\n" +
				"\n" +
				"If you identify that no action is required, it means the alert misfired, meaning the action to be taken here is to adjust the alert tirggering mechanism so that it only fires when it is actionable. If you find the alert signal useful, despite not being actionable, then the signal can be moved to a dashboard.\n" +
				"\n" +
				"Below are links referring to all alerts grouped. You must work until all of them are resolved.\n" +
				"\n" +
				"Currently firing alerts for this incident:\n" +
				"https://paymentsense.grafana.net/alerting/list?dataSource=DATASOURCE_NAME&queryString=tenant%3Dexample,urgency%3Dlow,label1%3Dvalue+%241,label2%3Dvalue+%242,&ruleType=alerting&alertState=firing\n" +
				"\n" +
				"History of alerts for this incident:\n" +
				"https://paymentsense.grafana.net/d/luyBQ9Y7z/?orgId=1&var-data_source=DATASOURCE_NAME&var-tenant=example&var-urgency=low&var-label=label1%7C%3D%7Cvalue+%241&var-label=label2%7C%3D%7Cvalue+%242&",
		},
		// dojo.documentation.unknown_urgency
		{
			title: "dojo.documentation.unknown_urgency",
			in:    `{{ template "dojo.documentation.unknown_urgency" . }}`,
			data: Data{
				GroupLabels: KV{
					"label1": "value $1",
					"label2": "value $2",
				},
				CommonLabels: KV{
					"tenant":  "example",
					"urgency": "low",
				},
			},
			exp: "An alert without a label urgency set to either high or low fired. The worst is assumed here: that the alert is of high urgency.\n" +
				"\n" +
				"This only happens when a misconfiguration happened on this alert, so it needs fixing. There are two actions required.\n" +
				"\n" +
				"The immediate action, is to evaluate the real urgency of the firing alert(s) and work on it accordingly.\n" +
				"\n" +
				"The secondary action, is to fix the alert configuration so that it fires with a correctly defined urgency next time.\n" +
				"\n" +
				"Below are links referring to all alerts grouped. You must work until all of them are resolved.\n" +
				"\n" +
				"Currently firing alerts for this incident:\n" +
				"https://paymentsense.grafana.net/alerting/list?dataSource=DATASOURCE_NAME&queryString=tenant%3Dexample,urgency%3Dlow,label1%3Dvalue+%241,label2%3Dvalue+%242,&ruleType=alerting&alertState=firing\n" +
				"\n" +
				"History of alerts for this incident:\n" +
				"https://paymentsense.grafana.net/d/luyBQ9Y7z/?orgId=1&var-data_source=DATASOURCE_NAME&var-tenant=example&var-urgency=low&var-label=label1%7C%3D%7Cvalue+%241&var-label=label2%7C%3D%7Cvalue+%242&",
		},
	} {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			f := tmpl.ExecuteTextString
			got, err := f(commonTmpl+tc.in, tc.data)
			if tc.fail {
				require.NotNil(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.exp, got)
		})
	}
}
