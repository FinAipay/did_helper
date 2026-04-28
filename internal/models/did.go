package models

import (
	"time"
)

// DID document structure - follows W3C DID Core v1.0 specification and FinAI DID rules
type DIDDocument struct {
	// JSON-LD context (required)
	Context interface{} `json:"@context"`

	// DID identifier (required)
	ID string `json:"id"`

	// Controller (optional)
	Controller interface{} `json:"controller,omitempty"`

	// Also known as (optional)
	AlsoKnownAs []string `json:"alsoKnownAs,omitempty"`

	// Verification methods (optional)
	VerificationMethod []*VerificationMethod `json:"verificationMethod,omitempty"`

	// Authentication (optional) - references verificationMethod
	Authentication []interface{} `json:"authentication,omitempty"`

	// Assertion method (optional) - references verificationMethod
	AssertionMethod []interface{} `json:"assertionMethod,omitempty"`

	// Key agreement (optional) - references verificationMethod
	KeyAgreement []interface{} `json:"keyAgreement,omitempty"`

	// Capability invocation (optional) - references verificationMethod
	CapabilityInvocation []interface{} `json:"capabilityInvocation,omitempty"`

	// Capability delegation (optional) - references verificationMethod
	CapabilityDelegation []interface{} `json:"capabilityDelegation,omitempty"`

	// Service endpoints (optional)
	Service []*Service `json:"service,omitempty"`

	// DID document metadata (optional)
	DIDDocumentMetadata *DocumentMetadata `json:"didDocumentMetadata,omitempty"`
}
type Service struct {
	ID              string      `json:"id"`                    // Service endpoint ID
	Type            string      `json:"type"`                  // Service endpoint type
	ServiceEndpoint interface{} `json:"serviceEndpoint"`       // Service endpoint URL or object
	Description     string      `json:"description,omitempty"` // Description (optional)

	// Metadata
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}
type ProofType string

// VerificationMethod verification method structure - follows FinAI DID rules
type VerificationMethod struct {
	// Required fields
	ID         string `json:"id"`         // Key identifier (e.g., #key-1 or full DID)
	Type       string `json:"type"`       // Key type
	Controller string `json:"controller"` // Controller DID

	// Public key fields (choose one based on type)
	EthereumAddress    string `json:"ethereumAddress,omitempty"`    // EVM address
	PublicKeyMultibase string `json:"publicKeyMultibase,omitempty"` // Multibase encoded public key
	PublicKeyBase58    string `json:"publicKeyBase58,omitempty"`    // Base58 encoded public key
	PublicKeyBase64    string `json:"publicKeyBase64,omitempty"`    // Base64 encoded public key
	PublicKeyPem       string `json:"publicKeyPem,omitempty"`       // PEM format public key
	PublicKeyHex       string `json:"publicKeyHex,omitempty"`       // Hex encoded public key

	// FinAI extension fields
	FinAIChain     string `json:"finai:chain,omitempty"`     // Blockchain type
	FinAINetwork   string `json:"finai:network,omitempty"`   // Network type
	FinAIPurpose   string `json:"finai:purpose,omitempty"`   // Key purpose
	FinAIIsPrimary bool   `json:"finai:isPrimary,omitempty"` // Whether it's the primary key

	// Metadata
	CreatedAt time.Time  `json:"createdAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`

	// Proof data (used to verify ownership)
	ProofType ProofType              `json:"proofType,omitempty"`
	ProofData map[string]interface{} `json:"proofData,omitempty"`
}
type DocumentMetadata struct {
	Created     string `json:"created"`               // Creation time (ISO8601)
	Updated     string `json:"updated,omitempty"`     // Update time (ISO8601)
	VersionID   string `json:"versionId,omitempty"`   // Version ID (UUID v4)
	HashLink    string `json:"hl,omitempty"`          // Document content hash (Hashlink)
	Deactivated bool   `json:"deactivated,omitempty"` // Whether deactivated

	// Extension fields
	EquivalentID []string `json:"equivalentId,omitempty"`
	CanonicalID  string   `json:"canonicalId,omitempty"`
}

// DIDMetadata DID document metadata
type DIDMetadata struct {
	VersionID   string    `json:"versionId,omitempty"`   // Version ID
	Created     time.Time `json:"created,omitempty"`     // Creation time
	Updated     time.Time `json:"updated,omitempty"`     // Update time
	Deactivated bool      `json:"deactivated,omitempty"` // Whether deactivated
	NextUpdate  time.Time `json:"nextUpdate,omitempty"`  // Next update time
}

type UpdateDIDRequest struct {
	AddKeys        []*VerificationMethod  `json:"addKeys,omitempty"`
	RevokeKeys     []string               `json:"revokeKeys,omitempty"`
	AddServices    []*Service             `json:"addServices,omitempty"`
	RemoveServices []string               `json:"removeServices,omitempty"`
	UpdateMetadata map[string]interface{} `json:"updateMetadata,omitempty"`
}

// Constant definitions
const (
	// W3C DID Core standard Verification Relationships
	PurposeAuthentication       = "authentication"       // Authentication
	PurposeAssertionMethod      = "assertionMethod"      // Assertion method
	PurposeKeyAgreement         = "keyAgreement"         // Key agreement
	PurposeCapabilityInvocation = "capabilityInvocation" // Capability invocation
	PurposeCapabilityDelegation = "capabilityDelegation" // Capability delegation
)

// Key type constants
const (
	KeyTypeEthereum = "EthereumAddressVerificationKey2020" // EVM key
	KeyTypeSolana   = "Ed25519VerificationKey2020"         // Solana key
	KeyTypeX25519   = "X25519KeyAgreementKey2019"          // X25519 key
)

// Entity type constants
const (
	EntityTypeUsers    = "users"    // Individual users
	EntityTypeAgents   = "agents"   // AI Agent
	EntityTypeDevices  = "devices"  // IoT devices
	EntityTypeServices = "services" // Service endpoints
	EntityTypeOrgs     = "orgs"     // Organizations/DAO
	EntityTypeAssets   = "assets"   // Digital assets
)

// Service type constants
const (
	ServiceTypeMetadata   = "FinAIEntityMetadata"     // Entity metadata
	ServiceTypePayment    = "FinAIPaymentService"     // Payment service
	ServiceTypeReputation = "FinAIReputationService"  // Reputation query
	ServiceTypeMessaging  = "FinAIEncryptedMessaging" // Encrypted messaging
	ServiceTypeTelemetry  = "FinAITelemetryService"   // Device telemetry
	ServiceTypeAPI        = "FinAIServiceAPI"         // API service
)

var DIDContexts = []string{
	"https://www.w3.org/ns/did/v1",
}

// NewDIDDocument creates a new DID document
func NewDIDDocument(entityType, entityId string) (*DIDDocument, error) {
	// Build DID: did:web:finai.network:{entityType}:{entityId}
	did := FormatDID(entityType, entityId)
	now := time.Now().Format(time.RFC3339)
	return &DIDDocument{
		Context: DIDContexts,
		ID:      did,
		DIDDocumentMetadata: &DocumentMetadata{
			Created:     now,
			Updated:     now,
			Deactivated: false,
		},
		VerificationMethod:   make([]*VerificationMethod, 0),
		Service:              make([]*Service, 0),
		Authentication:       make([]interface{}, 0),
		AssertionMethod:      make([]interface{}, 0),
		KeyAgreement:         make([]interface{}, 0),
		CapabilityInvocation: make([]interface{}, 0),
		CapabilityDelegation: make([]interface{}, 0),
	}, nil
}

// FormatDID formats DID string
func FormatDID(entityType, entityId string) string {
	return "did:finai:" + entityType + ":" + entityId
}

// AddVerificationMethod adds verification method
func (doc *DIDDocument) AddVerificationMethod(vm *VerificationMethod) {
	doc.VerificationMethod = append(doc.VerificationMethod, vm)
}

// AddToAuthentication adds to authentication reference
func (doc *DIDDocument) AddToAuthentication(keyID string) {
	doc.Authentication = append(doc.Authentication, keyID)
}

// AddToAssertionMethod adds to assertion method reference
func (doc *DIDDocument) AddToAssertionMethod(keyID string) {
	doc.AssertionMethod = append(doc.AssertionMethod, keyID)
}

// AddToKeyAgreement adds to key agreement reference
func (doc *DIDDocument) AddToKeyAgreement(keyID string) {
	doc.KeyAgreement = append(doc.KeyAgreement, keyID)
}

// AddToCapabilityInvocation adds to capability invocation reference
func (doc *DIDDocument) AddToCapabilityInvocation(keyID string) {
	doc.CapabilityInvocation = append(doc.CapabilityInvocation, keyID)
}

// AddToCapabilityDelegation adds to capability delegation reference
func (doc *DIDDocument) AddToCapabilityDelegation(keyID string) {
	doc.CapabilityDelegation = append(doc.CapabilityDelegation, keyID)
}

// AddService adds service endpoint
func (doc *DIDDocument) AddService(service *Service) *DIDDocument {
	doc.Service = append(doc.Service, service)
	return doc
}
