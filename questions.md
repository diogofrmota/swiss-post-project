1. do i need these?

finalizers: # used to prevents orphaned resources
    - resources-finalizer.argocd.argoproj.io

2. will this work? for the domain diogomota.com

spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: diogofrmota@gmail.com
    privateKeySecretRef:
      name: letsencrypt-prod-account-key
    solvers:
      - http01:
          ingress:
            ingressClassName: cilium
