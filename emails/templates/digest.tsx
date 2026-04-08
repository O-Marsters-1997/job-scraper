import { Html, Body, Container, Heading, Text } from "@react-email/components";
import { JobCard } from "../components/job-card";

export default function DigestEmail() {
  return (
    <Html lang="en">
      <Body style={bodyStyle}>
        <Container style={containerStyle}>
          <Heading style={headingStyle}>Job Digest</Heading>
          <Text style={subtitleStyle}>New jobs since your last digest</Text>

          {"{{range .Jobs}}"}
          <JobCard
            titleDirective={"{{.Title}}"}
            companyDirective={"{{.Company}}"}
            locationDirective={"{{.Location}}"}
            urlDirective={"{{.URL}}"}
          />
          {"{{end}}"}
        </Container>
      </Body>
    </Html>
  );
}

const bodyStyle = {
  backgroundColor: "#f9fafb",
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
};

const containerStyle = {
  maxWidth: "600px",
  margin: "0 auto",
  backgroundColor: "#ffffff",
  padding: "32px",
  borderRadius: "8px",
};

const headingStyle = {
  fontSize: "24px",
  fontWeight: "700",
  color: "#111827",
  margin: "0 0 8px",
};

const subtitleStyle = {
  fontSize: "14px",
  color: "#6b7280",
  margin: "0 0 24px",
};
