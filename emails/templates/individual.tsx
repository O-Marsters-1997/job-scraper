import { Html, Body, Container, Heading } from "@react-email/components";
import { Tailwind } from "@react-email/tailwind";
import { JobCard, type JobCardProps } from "../components/job-card";

const sampleJob: JobCardProps = {
  title: "Senior Software Engineer",
  company: "Acme Corp",
  location: "London, UK",
  url: "https://example.com/job/1",
  remuneration: "£80,000 – £100,000",
};

export default function IndividualEmail({
  title = sampleJob.title,
  company = sampleJob.company,
  location = sampleJob.location,
  url = sampleJob.url,
  remuneration = sampleJob.remuneration,
}: Partial<JobCardProps> = {}) {
  return (
    <Tailwind>
      <Html lang="en">
        <Body className="bg-gray-50 font-sans">
          <Container className="max-w-[600px] mx-auto bg-white p-8 rounded-lg">
            <Heading className="text-2xl font-bold text-gray-900 m-0 mb-6">
              New Job Posted
            </Heading>
            <JobCard
              title={title}
              company={company}
              location={location}
              url={url}
              remuneration={remuneration}
            />
          </Container>
        </Body>
      </Html>
    </Tailwind>
  );
}
