#!/bin/bash

# Define the name of the Nginx pod
POD_NAME="nginx-pod"

while true; do
  # Create a YAML file for the Nginx pod
  cat <<EOF > nginx-pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: $POD_NAME
spec:
  containers:
  - name: nginx-container
    image: nginx:latest
EOF

  # Apply the YAML file to create the pod
  kubectl apply -f nginx-pod.yaml

  # Wait for the pod to be running
  echo "Waiting for the pod to be in the 'Running' state..."
  kubectl wait --for=condition=Ready pod/$POD_NAME

  # Display pod information
  kubectl describe pod/$POD_NAME

  # Delete the pod
  kubectl delete pod/$POD_NAME

  # Clean up the YAML file
  rm nginx-pod.yaml

  echo "Nginx pod deleted."

  # Sleep for 2 minutes (every other minute)
#   sleep 120
done
