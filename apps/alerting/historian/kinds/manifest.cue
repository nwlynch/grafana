package kinds

manifest: {
	appName:       "alerting-historian"
	groupOverride: "historian.alerting.grafana.app"
	versions: {
		"v0alpha1": v0alpha1
	}
}

v0alpha1: {
    kinds: [dummyv0alpha1]

    routes: {
        namespaced: {
            "/alertstate/query": {
                "POST": {
                    request: {
                        body: #AlertStateQuery
                    }
                    response: {
                        entries: [...#AlertStateEntry]
                    }
                    responseMetadata: typeMeta: false
                }
            }
        }
    }
}

#AlertStateQuery: {
    from?: int64
    to?: int64
    limit?: int
    ruleUID?: string
    dashboardUID?: string
    panelID?: int64
    previous?: #State
    current?: #State
    labels?: {
        [string]: string
    }
}

#AlertStateEntry: {
    timestamp: int64
    line: string
}

#State: "normal" | "alerting" | "pending" | "nodata" | "error" | "recovering" @cuetsy(kind="enum",memberNames="Normal|Alerting|Pending|NoData|Error|Recovering")

dummyv0alpha1: {
    kind: "Dummy"
    schema: {
        // Spec is the schema of our resource. The spec should include all the user-editable information for the kind.
        spec: {
            dummyField: int
        }
    }
}