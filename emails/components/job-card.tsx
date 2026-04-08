import { Section, Heading, Text, Link, Hr } from "@react-email/components";

interface JobCardProps {
  titleDirective: string;
  companyDirective: string;
  locationDirective: string;
  urlDirective: string;
}

export function JobCard({
  titleDirective,
  companyDirective,
  locationDirective,
  urlDirective,
}: JobCardProps) {
  return (
    <Section style={sectionStyle}>
      <Heading as="h2" style={titleStyle}>
        {titleDirective}
      </Heading>
      <Text style={metaStyle}>
        {companyDirective} &middot; {locationDirective}
      </Text>
      <Link href={urlDirective} style={linkStyle}>
        View Job &rarr;
      </Link>
      <Hr style={hrStyle} />
    </Section>
  );
}

const sectionStyle = { padding: "16px 0" };
const titleStyle = {
  fontSize: "18px",
  fontWeight: "600",
  color: "#111827",
  margin: "0 0 4px",
};
const metaStyle = { fontSize: "14px", color: "#6b7280", margin: "0 0 8px" };
const linkStyle = {
  fontSize: "14px",
  color: "#2563eb",
  textDecoration: "none",
};
const hrStyle = { borderColor: "#e5e7eb", margin: "16px 0 0" };
