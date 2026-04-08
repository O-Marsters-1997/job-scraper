import type { PropsWithChildren } from "react";
import { Html, Body, Container, Heading, Text } from "@react-email/components";
import { Tailwind } from "@react-email/tailwind";
import { JobCard, type JobCardProps } from "../components/job-card";

const sampleJobs: JobCardProps[] = [
  {
    title: "Senior Software Engineer",
    company: "Acme Corp",
    location: "London, UK",
    url: "https://example.com/job/1",
    remuneration: "£80,000 – £100,000",
  },
  {
    title: "Staff Engineer",
    company: "Widget Inc",
    location: "Remote",
    url: "https://example.com/job/2",
  },
  {
    title: "Principal Engineer",
    company: "Startup Co",
    location: "Manchester, UK",
    url: "https://example.com/job/3",
    remuneration: "£120,000",
  },
];

interface DigestProps {
  jobs?: JobCardProps[];
}

export default function DigestEmail({
  children,
  jobs = sampleJobs,
}: PropsWithChildren<DigestProps>) {
  return (
    <Tailwind>
      <Html lang="en">
        <Body className="bg-gray-50 font-sans">
          <Container className="max-w-[600px] mx-auto bg-white p-8 rounded-lg">
            <Heading className="text-2xl font-bold text-gray-900 m-0 mb-2">
              Job Digest
            </Heading>
            <Text className="text-sm text-gray-500 m-0 mb-6">
              New jobs since your last digest
            </Text>
            {children ?? jobs.map((job, i) => <JobCard key={i} {...job} />)}
          </Container>
        </Body>
      </Html>
    </Tailwind>
  );
}
