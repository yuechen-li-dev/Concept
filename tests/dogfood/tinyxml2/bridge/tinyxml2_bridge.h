#ifndef CONCEPT_TINYXML2_BRIDGE_H
#define CONCEPT_TINYXML2_BRIDGE_H

typedef struct ConceptXmlStats {
    int children;
    int status;
} ConceptXmlStats;

#ifdef __cplusplus
#define CONCEPT_XML_NOEXCEPT noexcept
extern "C" {
#else
#define CONCEPT_XML_NOEXCEPT
#endif
unsigned char* ConceptXmlCreate(void) CONCEPT_XML_NOEXCEPT;
int ConceptXmlIsNull(unsigned char* document) CONCEPT_XML_NOEXCEPT;
int ConceptXmlParse(unsigned char* document) CONCEPT_XML_NOEXCEPT;
int ConceptXmlChildCount(unsigned char* document) CONCEPT_XML_NOEXCEPT;
ConceptXmlStats ConceptXmlGetStats(unsigned char* document) CONCEPT_XML_NOEXCEPT;
ConceptXmlStats ConceptXmlRoundTripStats(ConceptXmlStats stats) CONCEPT_XML_NOEXCEPT;
void ConceptXmlDestroy(unsigned char* document) CONCEPT_XML_NOEXCEPT;
#ifdef __cplusplus
}
#endif
#undef CONCEPT_XML_NOEXCEPT
#endif
