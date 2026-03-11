// PageMeta — Manages document head meta tags for SEO and social sharing
import { Helmet } from 'react-helmet-async';

const SITE_NAME = '대일외고 장학회';
const SITE_BASE_URL = import.meta.env.VITE_SITE_BASE_URL ?? 'https://daeilfoundation.or.kr';
const DEFAULT_DESC = '대일외고 동문 장학회 — 재학생 장학금 지원과 동문 네트워크를 운영합니다.';
const DEFAULT_OG_IMAGE = `${SITE_BASE_URL}/logo.png`;

interface ArticleData {
  headline: string;
  publishedAt: string;
}

interface PageMetaProps {
  title?: string;
  description?: string;
  ogImage?: string;
  canonicalPath?: string;
  noIndex?: boolean;
  ogType?: 'website' | 'article';
  articleData?: ArticleData;
}

export function PageMeta({
  title,
  description = DEFAULT_DESC,
  ogImage = DEFAULT_OG_IMAGE,
  canonicalPath,
  noIndex = false,
  ogType = 'website',
  articleData,
}: PageMetaProps) {
  const pageTitle = title ? `${title} — ${SITE_NAME}` : SITE_NAME;
  const canonicalURL = canonicalPath ? `${SITE_BASE_URL}${canonicalPath}` : undefined;
  const ogImageURL = ogImage.startsWith('/') ? `${SITE_BASE_URL}${ogImage}` : ogImage;

  const articleJsonLd = articleData
    ? JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'NewsArticle',
        headline: articleData.headline,
        datePublished: articleData.publishedAt,
        publisher: {
          '@type': 'Organization',
          name: SITE_NAME,
          logo: DEFAULT_OG_IMAGE,
        },
      })
    : null;

  return (
    <Helmet>
      <title>{pageTitle}</title>
      {noIndex && <meta name="robots" content="noindex, nofollow" />}
      <meta name="description" content={description} />
      {canonicalURL && <link rel="canonical" href={canonicalURL} />}
      <meta property="og:type" content={ogType} />
      <meta property="og:site_name" content={SITE_NAME} />
      <meta property="og:title" content={pageTitle} />
      <meta property="og:description" content={description} />
      {canonicalURL && <meta property="og:url" content={canonicalURL} />}
      <meta property="og:image" content={ogImageURL} />
      <meta property="og:locale" content="ko_KR" />
      <meta name="twitter:card" content="summary_large_image" />
      <meta name="twitter:title" content={pageTitle} />
      <meta name="twitter:description" content={description} />
      <meta name="twitter:image" content={ogImageURL} />
      {articleJsonLd && (
        <script type="application/ld+json">{articleJsonLd}</script>
      )}
    </Helmet>
  );
}
