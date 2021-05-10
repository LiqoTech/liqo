package identityManager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	"github.com/liqotech/liqo/pkg/discovery"
)

// CreateIdentity creates a new key and a new csr to be used as an identity to authenticate with a remote cluster.
func (certManager *certificateIdentityManager) CreateIdentity(remoteClusterID string) (*v1.Secret, error) {
	namespace, err := certManager.namespaceManager.GetNamespace(remoteClusterID)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return certManager.createIdentityInNamespace(remoteClusterID, namespace.Name)
}

// get the CertificateSigningRequest for a remote cluster.
func (certManager *certificateIdentityManager) GetSigningRequest(remoteClusterID string) ([]byte, error) {
	secret, err := certManager.getSecret(remoteClusterID)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	csrBytes, ok := secret.Data[csrSecretKey]
	if !ok {
		err = fmt.Errorf("csr not found in secret %v/%v for clusterID %v", secret.Namespace, secret.Name, remoteClusterID)
		klog.Error(err)
		return nil, err
	}

	return csrBytes, nil
}

// store the certificate issued by a remote authority for the specified remoteClusterID.
func (certManager *certificateIdentityManager) StoreCertificate(remoteClusterID string, certificate []byte) error {
	secret, err := certManager.getSecret(remoteClusterID)
	if err != nil {
		klog.Error(err)
		return err
	}

	if secret.Labels == nil {
		secret.Labels = map[string]string{}
	}
	secret.Labels[certificateAvailableLabel] = "true"

	secret.Data[certificateSecretKey] = certificate
	if _, err = certManager.client.CoreV1().Secrets(secret.Namespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
		klog.Error(err)
		return err
	}
	return nil
}

// retrieve the identity secret given the clusterID.
func (certManager *certificateIdentityManager) getSecret(remoteClusterID string) (*v1.Secret, error) {
	namespace, err := certManager.namespaceManager.GetNamespace(remoteClusterID)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			localIdentitySecretLabel: "true",
			discovery.ClusterIDLabel: remoteClusterID,
		},
	}
	secretList, err := certManager.client.CoreV1().Secrets(namespace.Name).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	secrets := secretList.Items
	if nItems := len(secrets); nItems == 0 {
		// create a new one
		return certManager.createIdentityInNamespace(remoteClusterID, namespace.Name)
	}

	// sort by reverse certificate expire time
	sort.Slice(secrets, func(i, j int) bool {
		time1 := getExpireTime(&secretList.Items[i])
		time2 := getExpireTime(&secretList.Items[j])
		return time1 > time2
	})

	// if there are multiple secrets, get the one with the certificate that will expire last
	return &secrets[0], nil
}

// generate a key and a certificate signing request.
func (certManager *certificateIdentityManager) createCSR() (keyBytes []byte, csrBytes []byte, err error) {
	key, err := rsa.GenerateKey(rand.Reader, keyLength)
	if err != nil {
		klog.Error(err)
		return nil, nil, err
	}

	subj := pkix.Name{
		CommonName:   certManager.localClusterID.GetClusterID(),
		Organization: []string{defaultOrganization},
	}
	rawSubj := subj.ToRDNSequence()

	asn1Subj, err := asn1.Marshal(rawSubj)
	if err != nil {
		klog.Error(err)
		return nil, nil, err
	}

	template := x509.CertificateRequest{
		RawSubject:         asn1Subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	csrBytes, err = x509.CreateCertificateRequest(rand.Reader, &template, key)
	if err != nil {
		klog.Error(err)
		return nil, nil, err
	}
	csrBytes = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	})

	keyBytes = x509.MarshalPKCS1PrivateKey(key)
	keyBytes = pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	})
	return keyBytes, csrBytes, nil
}

// create a new key and a new csr to be used as an identity to authenticate with a remote cluster in a given namespace.
func (certManager *certificateIdentityManager) createIdentityInNamespace(remoteClusterID string, namespace string) (*v1.Secret, error) {
	key, csrBytes, err := certManager.createCSR()
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: strings.Join([]string{identitySecretRoot, ""}, "-"),
			Namespace:    namespace,
			Labels: map[string]string{
				localIdentitySecretLabel: "true",
				discovery.ClusterIDLabel: remoteClusterID,
			},
			Annotations: map[string]string{
				// one year starting from now
				certificateExpireTimeAnnotation: fmt.Sprintf("%v", time.Now().AddDate(1, 0, 0).Unix()),
			},
		},
		Data: map[string][]byte{
			privateKeySecretKey: key,
			csrSecretKey:        csrBytes,
		},
	}

	return certManager.client.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
}

// read the expire time from the annotations of the secret.
func getExpireTime(secret *v1.Secret) int64 {
	now := time.Now().Unix()
	if secret.Annotations == nil {
		klog.Warningf("annotation %v not found in secret %v/%v", certificateExpireTimeAnnotation, secret.Namespace, secret.Name)
		return now
	}

	timeStr, ok := secret.Annotations[certificateExpireTimeAnnotation]
	if !ok {
		klog.Warningf("annotation %v not found in secret %v/%v", certificateExpireTimeAnnotation, secret.Namespace, secret.Name)
		return now
	}

	if n, err := strconv.ParseInt(timeStr, 10, 64); err != nil {
		klog.Warning(err)
		return now
	} else {
		return n
	}
}
