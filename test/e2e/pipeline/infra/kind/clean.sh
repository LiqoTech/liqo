# Cleaning all remaning clusters
# Export KIND path to the rest of the pipeline
KIND="${BINDIR}/kind"

${KIND} delete clusters --all