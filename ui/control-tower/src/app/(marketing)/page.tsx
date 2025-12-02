import {
  HeroSection,
  FeatureGrid,
  SocialProof,
  PricingTable,
  CTASection,
} from "@/components/marketing";

export default function HomePage() {
  return (
    <div className="page-transition">
      <HeroSection />
      <FeatureGrid />
      <SocialProof />
      <PricingTable />
      <CTASection />
    </div>
  );
}
