# Create a directory for your certificates
mkdir -p certs
cd certs

# Generate a private key
openssl genrsa -out localhost.key 2048

# Generate a self-signed certificate valid for 365 days
# The Common Name (CN) should match the hostname you will use (e.g., "localhost")
openssl req -new -x509 -sha256 -key localhost.key -out localhost.crt -days 365 -subj "/CN=localhost"
