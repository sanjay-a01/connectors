package providers

const Zoho Provider = "zoho"

func init() {
	// Zoho configuration
	SetInfo(Zoho, ProviderInfo{
		DisplayName: "Zoho",
		AuthType:    Oauth2,
		BaseURL:     "https://www.zohoapis.com",
		Oauth2Opts: &Oauth2Opts{
			GrantType:                 AuthorizationCode,
			AuthURL:                   "https://accounts.zoho.com/oauth/v2/auth",
			TokenURL:                  "https://accounts.zoho.com/oauth/v2/token",
			ExplicitScopesRequired:    true,
			ExplicitWorkspaceRequired: false,
			TokenMetadataFields: TokenMetadataFields{
				WorkspaceRefField: "api_domain",
				ScopesField:       "scope",
			},
		},
		Support: Support{
			BulkWrite: BulkWriteSupport{
				Insert: false,
				Update: false,
				Upsert: false,
				Delete: false,
			},
			Proxy:     true,
			Read:      false,
			Subscribe: false,
			Write:     false,
		},
		Media: &Media{
			DarkMode: &MediaTypeDarkMode{
				IconURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1722471872/media/zoho_1722471871.svg",
				LogoURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1722471872/media/zoho_1722471871.svg",
			},
			Regular: &MediaTypeRegular{
				IconURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1722471890/media/zoho_1722471890.svg",
				LogoURL: "https://res.cloudinary.com/dycvts6vp/image/upload/v1722471890/media/zoho_1722471890.svg",
			},
		},
	})
}
