#include "tinyxml2.h"
#include "tinyxml2_bridge.h"
#include <new>

// The upstream implementation is untouched. This narrow C ABI owns each
// XMLDocument until Destroy; exceptions never cross the foreign boundary.
extern "C" unsigned char* ConceptXmlCreate() noexcept {
    return reinterpret_cast<unsigned char*>(new (std::nothrow) tinyxml2::XMLDocument());
}

extern "C" int ConceptXmlIsNull(unsigned char* document) noexcept {
    return document == nullptr;
}

extern "C" int ConceptXmlParse(unsigned char* document) noexcept {
    if (!document) return -1;
    auto* xml = reinterpret_cast<tinyxml2::XMLDocument*>(document);
    return xml->Parse("<root><child/><child/></root>") == tinyxml2::XML_SUCCESS ? 0 : -1;
}

extern "C" int ConceptXmlChildCount(unsigned char* document) noexcept {
    if (!document) return -1;
    auto* xml = reinterpret_cast<tinyxml2::XMLDocument*>(document);
    auto* root = xml->RootElement();
    if (!root) return -1;
    int count = 0;
    for (auto* child = root->FirstChildElement(); child; child = child->NextSiblingElement()) ++count;
    return count;
}

extern "C" ConceptXmlStats ConceptXmlGetStats(unsigned char* document) noexcept {
    if (!document) return {-1, -1};
    auto* xml = reinterpret_cast<tinyxml2::XMLDocument*>(document);
    auto* root = xml->RootElement();
    if (!root) return {0, -1};
    int count = 0;
    for (auto* child = root->FirstChildElement(); child; child = child->NextSiblingElement()) ++count;
    return {count, 0};
}

extern "C" ConceptXmlStats ConceptXmlRoundTripStats(ConceptXmlStats stats) noexcept {
    return {stats.children + 1, stats.status + 2};
}

extern "C" void ConceptXmlDestroy(unsigned char* document) noexcept {
    delete reinterpret_cast<tinyxml2::XMLDocument*>(document);
}
