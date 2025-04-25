# test cloudformation template
Run sam validate on all template files
```
sam validate --lint -t deploy/db/template.yaml
sam validate --lint -t deploy/prayertexter/template.yaml
sam validate --lint -t deploy/statecontroller/template.yaml
```

# deploy to aws

On new installations, install stacks in this order:
1. db
2. prayertexter
3. stateresolver

Update/install specific stack:
```
cd deploy/<stack-name>
sam build
sam deploy --profile <local-aws-credential-profile>
```

If there are issues, you can also run guided builds and deployments:
```
sam build --guided
sam deploy --profile <local-aws-credential-profile> --guided
```