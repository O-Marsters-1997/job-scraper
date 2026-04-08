import { Section, Heading, Text, Link, Hr } from "@react-email/components";

export interface JobCardProps {
  title: string;
  company: string;
  location: string;
  url: string;
  remuneration?: string;
}

export function JobCard({
  title,
  company,
  location,
  url,
  remuneration,
}: JobCardProps) {
  return (
    <Section className="py-4">
      <Heading as="h2" className="text-lg font-semibold text-gray-900 m-0 mb-1">
        {title}
      </Heading>
      <Text className="text-sm text-gray-500 m-0 mb-2">
        {company} &middot; {location}
      </Text>
      {remuneration && (
        <Text className="text-sm text-gray-500 m-0 mb-2">{remuneration}</Text>
      )}
      <Link href={url} className="text-sm text-blue-600 no-underline">
        View Job &rarr;
      </Link>
      <Hr className="border-gray-200 mt-4" />
    </Section>
  );
}
