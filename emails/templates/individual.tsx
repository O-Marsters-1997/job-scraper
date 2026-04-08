import { Html, Body, Container, Heading } from "@react-email/components";
import { JobCard } from "../components/job-card";

// IndividualEmail renders the single-job notification template.
// Go template directives are embedded as literal strings.
export default function IndividualEmail() {
  return (
    <Html lang="en">
      <Body style={bodyStyle}>
        <Container style={containerStyle}>
          <Heading style={headingStyle}>New Job Posted</Heading>
          <JobCard
            titleDirective={"{{.Title}}"}
            companyDirective={"{{.Company}}"}
            locationDirective={"{{.Location}}"}
            urlDirective={"{{.URL}}"}
          />
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
  margin: "0 0 24px",
};
