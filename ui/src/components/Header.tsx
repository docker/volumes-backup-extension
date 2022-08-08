import { createDockerDesktopClient } from "@docker/extension-api-client";
import { QuestionAnswerOutlined } from "@mui/icons-material";
import { Grid, Link } from "@mui/material";
import Typography from "@mui/material/Typography/Typography";

const ddClient = createDockerDesktopClient();

const FEEDBACK_FORM_URL = "https://forms.gle/kYAwK34RUFXyAdfS7";

export const Header = () => (
  <>
    <Grid container gap={2} alignItems="center">
      <Typography variant="h3">Volumes Backup & Share</Typography>
      <Link
        href="#"
        onClick={() => {
          ddClient.host.openExternal(FEEDBACK_FORM_URL);
        }}
      >
        <Typography display="inline" variant="body2">
          Give Feedback
        </Typography>
        <QuestionAnswerOutlined
          fontSize="small"
          sx={{
            verticalAlign: "bottom",
            marginLeft: "0.25em",
          }}
        />
      </Link>
    </Grid>
    <Typography variant="body1" color="text.secondary" sx={{ mt: 2 }}>
      Backup and share Docker volumes effortlessly.
    </Typography>
  </>
);
